package storage

import (
	"fmt"
	"reflect"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util"
)

const (
	txnExpiredTime time.Duration = 5 * time.Second
)

func (r *replica) executeCmd(request meta.Request, eng engine.Engine) (meta.Response, error) {
	if !r.validateKeyRange(request.Header().Key) {
		return nil, errors.Trace(meta.NewRangeNotFoundError(r.rangeID))
	}
	switch req := request.(type) {
	case *meta.GetRequest:
		resp := &meta.GetResponse{}
		return resp, r.get(eng, req, resp)
	case *meta.PutRequest:
		resp := &meta.PutResponse{}
		return resp, r.put(req)
	case *meta.ScanRequest:
		resp := &meta.ScanResponse{}
		return resp, r.scan(eng, req, resp)
	case *meta.ReverseScanRequest:
		resp := &meta.ReverseScanResponse{}
		return resp, r.reverseScan(eng, req, resp)
	case *meta.BeginTransactionRequest:
		resp := &meta.BeginTransactionResponse{}
		resp.Txn = req.GetTxn()
		return resp, r.beginTransaction(req)
	case *meta.EndTransactionRequest:
		resp := &meta.EndTransactionResponse{}
		return resp, r.endTransaction(req)
	case *meta.HeartbeatTransactionRequest:
		resp := &meta.HeartbeatTransactionResponse{}
		return resp, r.heartbeatTransaction(req, resp)
	case *meta.ResolveIntentRequest:
		resp := &meta.ResolveIntentResponse{}
		return resp, r.resolveIntent(req)
	case *meta.PushTransactionRequest:
		resp := &meta.PushTransactionResponse{}
		return resp, r.pushTransaction(req, resp)
	case *meta.GetSplitKeyRequest:
		resp := &meta.GetSplitKeyResponse{}
		return resp, r.getSplitKey(eng, req, resp)
	case *meta.SplitRequest:
		resp := &meta.SplitResponse{}
		return resp, r.split(req, resp)
	case *meta.AddRaftConfRequest:
		resp := &meta.AddRaftConfResponse{}
		return resp, r.addRaftConf(req, resp)
	case *meta.DeleteRaftConfRequest:
		resp := &meta.DeleteRaftConfResponse{}
		return resp, r.deleteRaftConf(req, resp)
	case *meta.UpdateRangeRequest:
		resp := &meta.UpdateRangeResponse{}
		return resp, r.updateRange(req)
	case *meta.CompactLogRequest:
		resp := &meta.CompactLogResponse{}
		return resp, r.compactLog(req)
	case *meta.TransferLeaderRequest:
		resp := &meta.TransferLeaderResponse{}
		return resp, r.transferLeader(req)
	}

	log.Errorf("range:%v no found request type:%s", r.rangeID, reflect.TypeOf(request).String())
	return nil, errors.Errorf("no found request type:%v", reflect.TypeOf(request).String())
}

func (r *replica) get(eng engine.Engine, req *meta.GetRequest, resp *meta.GetResponse) error {
	key := req.Key
	txn := req.GetTxn()
	ts := req.GetTimestamp()
	name := txn.Name

	log.Debugf("[%d %d] txn:%s get key:%v", r.store.nodeID, r.rangeID, name, key)
	val, err := r.getInternal(eng, key, ts, txn)
	if err != nil {
		return errors.Trace(err)
	}

	if val == nil {
		val = &meta.Value{}
	}

	resp.KeyValue = meta.KeyValue{Key: key, Value: *val}
	resp.Timestamp = ts
	log.Debugf("[%d %d] %s get key:%q, value:%v", r.store.nodeID, r.rangeID, name, key, val.String())

	return nil
}

func (r *replica) pushTransaction(req *meta.PushTransactionRequest, resp *meta.PushTransactionResponse) error {
	key := req.PusheeKey
	pushType := req.PushType
	pusherTxn := req.GetTxn()
	ts := req.GetLatestTimestamp()

	log.Debugf("[%d %d] %s processPushTransaction", r.store.nodeID, r.rangeID, pusherTxn.Name)
	pusheeTxn, err := r.processPushTransaction(key, ts, pushType, pusherTxn)
	if pusheeTxn != nil {
		resp.PusheeTxn = pusheeTxn
	}
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (r *replica) processPushTransaction(key meta.Key, latestTimestamp meta.Timestamp, pushType meta.PushTxnType,
	pusherTxn *meta.Transaction) (*meta.Transaction, error) {
	//FIXME: can we hide pusherTxnTimestamp into pusherTxn object?
	log.Debugf("range:%v %s processPushTransaction key:%q latestTimestamp:%v pushType:%v pushertxn:%v",
		r.rangeID, pusherTxn.Name, key, latestTimestamp.String(), pushType.String(), pusherTxn.String())

	pusheeTxn, err := getTxnData(r.store.engine, key)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if pusheeTxn == nil {
		return nil, errors.Trace(meta.NewInvalidTransactionError(fmt.Sprintf("not found txnID:%q", key)))
	}

	nowTime := latestTimestamp.WallTime
	pusheeStatus := pusheeTxn.Status
	pusherTimestamp := pusherTxn.GetTimestamp()
	pusheeHeartbeat := pusheeTxn.LastHeartbeat
	log.Debugf("range:%v %s processPushTransaction, check pusheeTxnStatus:%v", r.rangeID, pusherTxn.Name, pusheeStatus)

	if pusheeStatus != meta.PENDING {
		return pusheeTxn, nil
	}

	log.Debugf("range:%v %s processPushTransaction, pushType:%v pusheeTxnTimestamp:%v pusherTxnTimestamp:%v",
		r.rangeID, pusherTxn.Name, pushType, pusheeTxn.Timestamp.String(), pusherTimestamp.String())

	if pushType == meta.PUSH_TIMESTAMP && pusheeTxn.Timestamp.Large(pusherTimestamp) {
		return pusheeTxn, nil
	}

	log.Debugf("range:%v %s processPushTransaction, check timeout itTimeout:%v", r.rangeID,
		pusherTxn.Name, nowTime-pusheeHeartbeat > int64(txnExpiredTime))
	if nowTime-pusheeHeartbeat > int64(txnExpiredTime) {
		pusheeTxn.Status = meta.ABORTED
		if err := putTxnData(r.store.engine, pusheeTxn, meta.TxnChangeTypePushAbort, nil); err != nil {
			return nil, errors.Trace(err)
		}

		return pusheeTxn, nil
	}

	if pusherTxn.Priority < pusheeTxn.Priority {
		return nil, errors.Trace(meta.NewTransactionAbortedError(pusherTxn))
	}

	if pushType == meta.PUSH_TIMESTAMP {
		pusheeTxn.Timestamp = pusherTimestamp.Next()
		log.Debugf("range:%v %s processPushTransaction, do PUSH_TIMESTAMP timestamp +1 key:%q pusheeTxnTimestamp:%v",
			r.rangeID, pusherTxn.Name, key, pusheeTxn.Timestamp.String())
		if err := putTxnData(r.store.engine, pusheeTxn, meta.TxnChangeTypePushTimestamp, nil); err != nil {
			return nil, errors.Trace(err)
		}

		return pusheeTxn, nil
	} else if pushType == meta.PUSH_ABORT {
		log.Debugf("range:%v %s processPushTransaction, do PUSH_ABORT", r.rangeID, pusherTxn.Name)
		pusheeTxn.Status = meta.ABORTED
		if err := putTxnData(r.store.engine, pusheeTxn, meta.TxnChangeTypePushAbort, nil); err != nil {
			return nil, errors.Trace(err)
		}
		return pusheeTxn, nil
	}

	return nil, errors.Trace(meta.NewTransactionAbortedError(pusherTxn))
}

func isNoneValue(val meta.Value) bool {
	return len(val.Bytes) == 0 && val.Checksum == 0 &&
		val.Timestamp == nil && val.Tag == 0
}

// put handle put request
func (r *replica) put(req *meta.PutRequest) error {
	key := req.Key
	txn := req.GetTxn()
	name := txn.Name
	val := req.Value
	ts := req.GetTimestamp()
	noneValue := isNoneValue(val)

	log.Debugf("[%d %d] %s put key:%q value:%v", r.store.nodeID, r.rangeID, name, key, val.String())
	metaData, origKeySize, origValSize, err := getMetaData(r.store.engine, key)
	if err != nil {
		return errors.Trace(err)
	}

	if metaData != nil {
		if metaData.GetTransactionID() == nil {
			if ts.Less(metaData.Timestamp) {
				log.Infof("[%d %d] %s put timestamp(%v) is too old, metaTimestamp:%v",
					r.store.nodeID, r.rangeID, name, ts.String(), metaData.Timestamp.String())
				return errors.Trace(meta.NewWriteTooOldError(ts, metaData.Timestamp))
			}
		} else {
			if util.Compare(metaData.TransactionID, txn.ID) != 0 {
				return errors.Trace(meta.NewWriteIntentError(txn, meta.PUSH_ABORT, metaData.TransactionID, key))
			}

			// if go these the reqTxn and the metaData.Transaction maybe the same transaction.
			// if timestamp not equal, then clear the data.
			if !metaData.Timestamp.Equal(ts) {
				if err = clearEngineData(r.store.engine, key, metaData.Timestamp); err != nil {
					return errors.Trace(err)
				}
			}
		}
	}

	if noneValue {
		if metaData == nil || (metaData != nil && metaData.Deleted) {
			return nil
		}
	}

	if err = r.putDataAndMeta(key, txn, val, noneValue, origKeySize, origValSize); err != nil {
		return errors.Trace(err)
	}
	log.Debugf("[%d %d] %s put end, key:%v txn:%v", r.store.nodeID, r.rangeID, name, key, txn.String())

	return nil
}

// putDataAndMeta do set key' metadata and key's mvcc value to Engine
func (r *replica) putDataAndMeta(key meta.Key, txn *meta.Transaction, val meta.Value, deleted bool, origKevSize, origValSize int) error {
	// put MetaData or update MetaData
	log.Debugf("range:%v putDataAndMeta key: %q, value: %v reqTxn: %v", r.rangeID, key, val, txn.String())
	if err := putMetaData(r.store.engine, key, txn, deleted, r.store.rangeStats[r.rangeID], origKevSize, origValSize); err != nil {
		return errors.Trace(err)
	}

	//put value or upate value
	if err := putEngineData(r.store.engine, key, txn, val, r.store.rangeStats[r.rangeID]); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (r *replica) beginTransaction(req *meta.BeginTransactionRequest) error {
	txn := req.GetTxn()
	name := txn.Name
	log.Debugf("range:%v %s beginTransactionRequest parpare to MVCCPutTxn", r.rangeID, name)
	if err := putTxnData(r.store.engine, txn, meta.TxnChangeTypeBegin, r.store.rangeStats[r.rangeID]); err != nil {
		return errors.Trace(err)
	}

	log.Debugf("range:%v %s beginTransactionRequest end", r.rangeID, name)
	return nil
}

func (r *replica) endTransaction(req *meta.EndTransactionRequest) error {
	txn := req.GetTxn()
	name := txn.Name

	log.Debugf("range:%v %s endTransactionRequest reqCommit:%v", r.rangeID, name, req.Commit)
	if req.Commit {
		txn.Status = meta.COMMITTED
	} else {
		txn.Status = meta.ABORTED
	}

	log.Debugf("range:%v %s endTransactionRequest prepare to MVCCPutTxn", r.rangeID, name)
	if err := putTxnData(r.store.engine, txn, meta.TxnChangeTypeEnd, nil); err != nil {
		return errors.Trace(err)
	}

	log.Debugf("range:%v %s endTransactionRequest end", r.rangeID, name)
	return nil
}

// heartbeatTransaction handle heartbeat transaction request
func (r *replica) heartbeatTransaction(req *meta.HeartbeatTransactionRequest, resp *meta.HeartbeatTransactionResponse) error {
	txn := req.GetTxn()
	name := txn.Name

	log.Debugf("[%d %d] %s heartbeatTransaction", r.store.nodeID, r.rangeID, name)
	if err := putTxnData(r.store.engine, txn, meta.TxnChangeTypeHeartbeat, nil); err != nil {
		if _, ok := errors.Cause(err).(*meta.TransactionStatusError); ok {
			resp.Txn = &meta.Transaction{ID: txn.ID, Status: meta.ABORTED}
			return nil
		}
		log.Errorf("[%d %d] %s heartbeatTransaction error:%s", r.store.nodeID, r.rangeID, name, errors.ErrorStack(err))
	}

	return nil
}

// resolveIntent handle resolve intent request
func (r *replica) resolveIntent(req *meta.ResolveIntentRequest) error {
	log.Debugf("[%d %d] txn:%s resolveIntentRequest begin", r.store.nodeID, r.rangeID, req.GetTxn().Name)
	if err := engine.MVCCResolveWriteIntent(r.store.engine, req.GetIntents(), r.store.rangeStats[r.rangeID]); err != nil {
		return errors.Trace(err)
	}
	log.Debugf("[%d %d] txn:%s resolveIntentRequest end", r.store.nodeID, r.rangeID, req.GetTxn().Name)
	return nil
}

func (r *replica) scan(eng engine.Engine, req *meta.ScanRequest, resp *meta.ScanResponse) error {
	kvs, err := r.scanInternal(eng, req.Key, req.EndKey, req.GetTxn(), req.GetTimestamp(), req.MaxResults, false)
	if err != nil {
		return errors.Trace(err)
	}
	resp.Rows = kvs

	return nil
}

func (r *replica) reverseScan(eng engine.Engine, req *meta.ReverseScanRequest, resp *meta.ReverseScanResponse) error {
	kvs, err := r.scanInternal(eng, req.Key, req.EndKey, req.GetTxn(), req.GetTimestamp(), req.MaxResults, true)
	if err != nil {
		return errors.Trace(err)
	}
	resp.Rows = kvs

	return nil
}

// scan resolve scanRequest
func (r *replica) scanInternal(snapshot engine.Engine, startKey, endKey meta.Key,
	reqTxn *meta.Transaction,
	ts meta.Timestamp,
	max int64,
	reverse bool) ([]meta.KeyValue, error) {
	txnName := reqTxn.Name
	keyValues := []meta.KeyValue{}

	log.Debugf("[%d %d] txn:%s scan startKey:%v endKey:%v maxResults:%v timestamp:%v",
		r.store.nodeID, r.rangeID, txnName, startKey.String(), endKey.String(), max, ts.String())

	if len(startKey) == 0 {
		return keyValues, errors.Trace(meta.NewInvalidKeyError())
	}

	var encKey, encEndKey meta.MVCCKey
	if reverse {
		encEndKey = meta.NewMVCCKey(startKey)
		encKey = meta.NewMVCCKey(endKey)
	} else {
		encEndKey = meta.NewMVCCKey(endKey)
		encKey = meta.NewMVCCKey(startKey)
	}

	log.Debugf("range:%v %s scan encKey: %v, encEndKey: %v\n", r.rangeID, txnName, encKey, encEndKey)

	// Get a new iterator.
	iter := snapshot.NewIterator()
	defer iter.Close()

	// Seeking for the first defined position.
	if reverse {
		iter.SeekReverse(encKey)
		if !iter.Valid() {
			return keyValues, iter.Error()
		}

		// If the key doesn't exist, the iterator is at the next key that does exist in the database.
		metaKey := iter.Key()
		if !metaKey.Less(encKey) {
			iter.Prev()
		}
	} else {
		iter.Seek(encKey)
	}

	if !iter.Valid() {
		return keyValues, iter.Error()
	}

	var cnt int64
	for {
		metaKey, encMetaKey, err := engine.GetMetaKey(iter, encEndKey, reverse)
		if err != nil {
			return keyValues, errors.Trace(err)
		}

		if metaKey == nil {
			log.Debugf("range:%v %s scan meta key is nil", r.rangeID, txnName)
			break
		}

		log.Debugf("range:%v %s scan getInternal metaKey:%v", r.rangeID, txnName, metaKey)
		value, err := r.getInternal(snapshot, metaKey, ts, reqTxn)
		if err != nil {
			return keyValues, errors.Trace(err)
		}

		if value != nil && len(value.Bytes) != 0 {
			kv := meta.KeyValue{Key: metaKey, Value: *value}
			keyValues = append(keyValues, kv)

			cnt++
			log.Debugf("range:%v %s scan kv:%v, resultCount:%v", r.rangeID, txnName, kv.String(), cnt)
			if cnt == max && max > 0 {
				break
			}
		}

		if reverse {
			// Seeking for the position of the given meta key.
			iter.Seek(encMetaKey)
			if iter.Valid() {
				// Move the iterator back, which gets us into the previous row (getMeta
				// moves us further back to the meta key).
				iter.Prev()
			}
		} else {
			log.Debugf("range:%v %s scan Seec metaKey Next", r.rangeID, txnName)
			iter.Seek(meta.NewMVCCKey(metaKey.Next()))
		}

		if !iter.Valid() {
			if err := iter.Error(); err != nil {
				return keyValues, errors.Trace(err)
			}
			break
		}
	}

	log.Debugf("[%d %d] %s scan resp rows:%v", r.store.nodeID, r.rangeID, txnName, keyValues)
	return keyValues, nil
}

func (r *replica) getInternal(eng engine.Engine, key meta.Key, ts meta.Timestamp, txn *meta.Transaction) (*meta.Value, error) {
	name := txn.Name

	metaData, _, _, err := getMetaData(eng, key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if metaData == nil {
		return nil, nil
	}

	log.Debugf("range:%v %s getInternal key:%q timestamp:%v, metaData: %v", r.rangeID, name, key, ts.String(), metaData)
	checkDel := false
	if metaData.TransactionID != nil {
		if util.Compare(metaData.TransactionID, txn.ID) != 0 {
			if !ts.Less(metaData.Timestamp) {
				log.Infof("[%d %d] %s getInternal the transaction is R/W conflict, oldTxn:%s", r.store.nodeID, r.rangeID, name, metaData.TransactionID)
				return nil, errors.Trace(meta.NewWriteIntentError(txn, meta.PUSH_TIMESTAMP, metaData.GetTransactionID(), key))
			}
		} else {
			checkDel = true
			ts = metaData.Timestamp
		}
	} else {
		if !ts.Less(metaData.Timestamp) {
			checkDel = true
			log.Debugf("range:%v %s getInternal need check deleted, request timestamp less metaData timestamp", r.rangeID, name)
		}
	}

	if checkDel && metaData.Deleted {
		log.Debugf("range:%v getInternal meta data is deleted\n", r.rangeID)
		return nil, nil
	}

	log.Debugf("range:%v %s getInternal prepare to getEngineData, key:%q timestamp:%v txn:%v",
		r.rangeID, name, key, ts.String(), txn.String())
	val, err := getEngineData(eng, key, ts)
	if err != nil {
		return nil, errors.Trace(err)
	}

	log.Debugf("range:%v %s getInternal debug Get key:%q  value:%v", r.rangeID, name, key, val.String())
	return val, nil
}

// putTxnData package of MVCCPutTxn to put a transaction object to engine
func putTxnData(eng engine.Engine, txn *meta.Transaction, changeType int, stats *meta.RangeStats) error {
	if err := engine.MVCCPutTxn(eng, txn, changeType, stats); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// getMetaData package of MVCCGetMeta
func getMetaData(eng engine.Engine, key meta.Key) (*engine.MVCCMetadata, int, int, error) {
	metaData, keySize, valSize, err := engine.MVCCGetMeta(eng, key)
	if err != nil {
		return nil, 0, 0, errors.Trace(err)
	}
	return metaData, keySize, valSize, nil
}

// getTxnData package of MVCCGetMeta to get a Transaction object data
func getTxnData(eng engine.Engine, key meta.Key) (*meta.Transaction, error) {
	txn, err := engine.MVCCGetTxn(eng, key)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return txn, nil
}

// getEngineData package of MVCCGet to get value
func getEngineData(eng engine.Engine, key meta.Key, timestamp meta.Timestamp) (*meta.Value, error) {
	val, _, _, err := engine.MVCCGet(eng, key, timestamp)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return val, nil
}

// putMetaData package of MVCCPutMeta to insert or update a metaData
func putMetaData(eng engine.Engine, key meta.Key, txn *meta.Transaction, deleted bool, stats *meta.RangeStats,
	origKeySize, origValSize int) error {
	if _, _, err := engine.MVCCPutMeta(eng, key, txn, deleted, stats, origKeySize, origValSize); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// putEngineData package of MVCCPut to insert a keyvalue or update a keyvalue
func putEngineData(eng engine.Engine, key meta.Key, txn *meta.Transaction,
	val meta.Value, stats *meta.RangeStats) error {
	if err := engine.MVCCPut(eng, key, txn, val, stats); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// clearEngineData package of MVCCClear to remove a key
func clearEngineData(eng engine.Engine, key meta.Key, timestamp meta.Timestamp) error {
	if err := engine.MVCCClear(eng, key, timestamp); err != nil {
		return errors.Trace(err)
	}
	return nil
}
