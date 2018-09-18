package engine

import (
	"bytes"
	"fmt"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

// MVCCGetMeta get metadata from engine
func MVCCGetMeta(engine Engine, key meta.Key) (*MVCCMetadata, int, int, error) {
	if len(key) == 0 {
		return nil, 0, 0, errors.Trace(meta.NewInvalidKeyError())
	}

	metaKey := meta.NewMVCCKey(key)
	val, err := engine.Get(metaKey)
	if err != nil {
		return nil, 0, 0, errors.Trace(meta.NewEngineGetDataError(key, "GetMeta failure"))
	}
	if val == nil {
		return nil, 0, 0, nil
	}

	metaData := &MVCCMetadata{}
	if err := metaData.Unmarshal(val); err != nil {
		return nil, 0, 0, errors.Annotatef(meta.NewUnmarshalDataError(val, err.Error()), "key:%q", metaKey)
	}

	log.Debugf("MVCCGetMetaData metaData:%v", metaData)
	return metaData, len(metaKey), len(val), nil
}

// MVCCGetTxn get transaction from engine
func MVCCGetTxn(engine Engine, key meta.Key) (*meta.Transaction, error) {
	if len(key) == 0 {
		return nil, errors.Trace(meta.NewInvalidKeyError())
	}

	txnKey := meta.NewMVCCKey(key)
	log.Debugf("MVCCGetTxn key:%v", txnKey)
	val, err := engine.Get(txnKey)
	if err != nil {
		return nil, errors.Trace(meta.NewEngineGetDataError(txnKey, err.Error()))
	}
	if val == nil {
		return nil, nil
	}

	txn := &meta.Transaction{}
	if err = txn.Unmarshal(val); err != nil {
		return nil, errors.Trace(meta.NewUnmarshalDataError(val, err.Error()))
	}

	return txn, nil
}

// MVCCGet returns the value for the key specified in the request,
// while satisfying the given timestamp condition. The key may contain
// arbitrary bytes. If no value for the key exists, or it has been
// deleted, returns nil for value.
//
// The values of multiple versions for the given key should
// be organized as follows:
// ...
// keyA : MVCCMetadata of keyA
// keyA_Timestamp_n : value of version_n
// keyA_Timestamp_n-1 : value of version_n-1
// ...
// keyA_Timestamp_0 : value of version_0
// keyB : MVCCMetadata of keyB
func MVCCGet(engine Engine, key meta.Key, ts meta.Timestamp) (*meta.Value, int, int, error) {
	if len(key) == 0 {
		return nil, 0, 0, errors.Trace(meta.NewInvalidKeyError())
	}

	log.Debugf("MVCCGet mvccGetInternal key:%s timestamp%v", key, ts)
	val, keySize, valSize, err := mvccGetInternal(engine, key, ts)
	if err != nil {
		return nil, 0, 0, errors.Trace(err)
	}

	return val, keySize, valSize, nil
}

func mvccGetInternal(engine Engine, key meta.Key, ts meta.Timestamp) (*meta.Value, int, int, error) {
	encKey := meta.NewMVCCTimeKey(key, ts)
	log.Debugf("mvccGetInternal latestKey:%v", encKey.String())

	// scans for the first key >= latestKey.
	iter := engine.NewIterator(false)
	defer iter.Close()

	iter.Seek(encKey)
	if !iter.Valid() {
		log.Errorf("getValue failure key:%q, ts:%v, encKey:%q", key, ts, encKey)
		return nil, 0, 0, nil
	}

	mvccKey := iter.Key()
	val := iter.Value()
	mvccKeySize := len(mvccKey)
	encKeySize := len(encKey)

	if val == nil {
		return nil, mvccKeySize, 0, nil
	}

	log.Debugf("mvccGetInternal latestKey:%v, found key: %v, value:%v", encKey.String(), mvccKey.String(), val)
	if mvccKeySize != encKeySize || !bytes.Equal(mvccKey[:mvccKeySize-meta.EncodeTimestampSize], encKey[:encKeySize-meta.EncodeTimestampSize]) {
		return nil, mvccKeySize, 0, nil
	}

	mvccValue := MVCCValue{}
	valSize := len(val)
	if err := mvccValue.Unmarshal(val); err != nil {
		return nil, mvccKeySize, valSize, errors.Trace(meta.NewUnmarshalDataError(val, err.Error()))
	}

	log.Debugf("mvccGetInternal get key:%v value:%v", key.String(), mvccValue.String())
	return mvccValue.Value, mvccKeySize, valSize, nil
}

// MVCCPutMeta used to Add a new MetaData or update a exist MetaData
func MVCCPutMeta(engine Engine, key meta.Key, txn *meta.Transaction, deleted bool, stats *meta.RangeStats, origKeySize int, origValSize int) (int, int, error) {
	if len(key) == 0 {
		return 0, 0, errors.Trace(meta.NewInvalidKeyError())
	}

	metaData := MVCCMetadata{
		TransactionID: txn.ID,
		Timestamp:     txn.GetTimestamp(),
		Deleted:       deleted,
	}

	metaKey := meta.NewMVCCKey(key)
	log.Debugf("MVCCPutMeta key:%q metaData:%v", key, metaData.String())

	data, err := metaData.Marshal()
	if err != nil {
		return 0, 0, errors.Trace(meta.NewMarshalDataError(err.Error()))
	}

	if err := engine.Put(metaKey, data); err != nil {
		return 0, 0, errors.Trace(meta.NewEnginePutDataError(metaKey, data, err.Error()))
	}

	if stats != nil {
		stats.UpdateRangeStats(origKeySize, origValSize, len(metaKey), len(data))
	}

	return len(metaKey), len(data), nil
}

// MVCCPutTxn used to put a TxnData.
func MVCCPutTxn(engine Engine, reqTxn *meta.Transaction, changeType int, stats *meta.RangeStats) error {
	key := meta.NewMVCCKey(reqTxn.ID)
	localTxn := *reqTxn
	localTxn.LastHeartbeat = localTxn.Timestamp.WallTime
	needUpdate := true

	log.Debugf("MVCCPutTxn timestamp:%v txnID:%q key:%q", reqTxn.Timestamp, reqTxn.ID, key)

	if changeType != meta.TxnChangeTypeBegin {
		txn, err := MVCCGetTxn(engine, reqTxn.ID)
		if err != nil {
			return errors.Trace(err)
		}
		if txn != nil {
			localTxn = *txn
			needUpdate = false
		}
	}

	if localTxn.Status == meta.COMMITTED {
		//nothing to do here
		return errors.Trace(meta.NewTransactionStatusError(localTxn.Clone(), "Transaction had COMMITTED", false))
	}

	if localTxn.Status == meta.ABORTED {
		//need client retry
		return errors.Trace(meta.NewTransactionStatusError(localTxn.Clone(), "Transaction had ABORTED", true))
	}

	//TODO: the algorithm here is if we commit with a lesser timestamp,
	// we use original one without retry, not sure about it's ok or not.
	if changeType == meta.TxnChangeTypePushTimestamp && reqTxn.Timestamp.Less(localTxn.Timestamp) {
		//attempt to push with lesser timestamp, return err here.
		return nil
	}

	if changeType == meta.TxnChangeTypeHeartbeat && reqTxn.LastHeartbeat > localTxn.LastHeartbeat {
		localTxn.LastHeartbeat = reqTxn.LastHeartbeat
		needUpdate = true
	}

	if localTxn.Timestamp.Less(reqTxn.Timestamp) {
		localTxn.Timestamp = reqTxn.Timestamp
		needUpdate = true
	}

	if changeType == meta.TxnChangeTypePushAbort || changeType == meta.TxnChangeTypeEnd {
		localTxn.Status = reqTxn.Status
		needUpdate = true
	}

	if !needUpdate {
		return nil
	}

	data, err := localTxn.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	if err = engine.Put(key, data); err != nil {
		return errors.Trace(meta.NewEnginePutDataError(key, data, err.Error()))
	}

	if stats != nil {
		stats.UpdateRangeStats(0, 0, len(key), len(data))
	}

	return nil
}

// MVCCPut put the mvcc value to Engine
func MVCCPut(engine Engine, key meta.Key, txn *meta.Transaction, value meta.Value, stats *meta.RangeStats) error {
	val := MVCCValue{Value: &value}
	if err := mvccPutInternal(engine, key, val, txn, stats); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// mvccPutInternal adds a new timestamped value to the specified key.
// If value is nil, creates a deletion tombstone value.
func mvccPutInternal(engine Engine, key meta.Key, val MVCCValue, txn *meta.Transaction, stats *meta.RangeStats) error {
	if txn == nil {
		return errors.Trace(meta.NewInvalidTransactionError(fmt.Sprintf("transaction is nil")))
	}

	mvccKey := meta.NewMVCCTimeKey(key, txn.GetTimestamp())
	data, err := val.Marshal()
	if err != nil {
		return errors.Trace(meta.NewMarshalDataError(fmt.Sprintf("%v", err)))
	}

	if err := engine.Put(mvccKey, data); err != nil {
		return errors.Trace(meta.NewEnginePutDataError(mvccKey, data, fmt.Sprintf("%v", err)))
	}

	if stats != nil {
		stats.UpdateRangeStats(0, 0, len(mvccKey), len(data))
	}

	log.Debugf("mvccPutInternal, put key data mvccKey:%s value:%v", mvccKey.String(), val.String())

	return nil
}

//MVCCResolveWriteIntent  we have three case here
//case 1: commit, set transaction id of meta equal to zero, if commit timestamp larger than local's, same as push timestamp
//case 2: rollback, clear the latest version of data key, and set meta to reflect the pervious version of data key, if doesn't existed, clear meta.
//case 3: push timestamp, exchange the latest key with new timestamp as part of it, also change the timestamp of meta.
func MVCCResolveWriteIntent(engine Engine, intents []meta.Intent, stats *meta.RangeStats) error {
	for _, intent := range intents {
		resolveTxn := intent.Txn
		resolveKey := intent.Key

		localMeta, origMetaKeySize, origMetaValSize, err := MVCCGetMeta(engine, resolveKey)
		if err != nil {
			log.Debugf("MVCCResolveWriteIntent GetMeta err:%v", errors.ErrorStack(err))
			continue
		}

		if localMeta == nil {
			log.Debugf("MVCCResolveWriteIntent localMeta is nil")
			continue
		}

		// check TransactionID need equal
		if bytes.Compare(localMeta.TransactionID, resolveTxn.ID) != 0 {
			log.Debugf("MVCCResolveWriteIntent received resolvedTxn ID %s doesn't match with local", resolveTxn.ID)
			continue
		}

		updatedTxn := &meta.Transaction{Timestamp: resolveTxn.Timestamp}
		switch resolveTxn.Status {
		case meta.PENDING:
			updatedTxn.ID = resolveTxn.ID
			err = mvccResolveIntentPending(engine, resolveKey, updatedTxn, localMeta,
				stats, origMetaKeySize, origMetaValSize)

		case meta.ABORTED:
			err = mvccResolveIntentAbort(engine, resolveKey, updatedTxn, localMeta,
				stats, origMetaKeySize, origMetaValSize)

		case meta.COMMITTED:
			err = mvccResolveIntentCommit(engine, resolveKey, updatedTxn, localMeta,
				stats, origMetaKeySize, origMetaValSize)
		}

		if err != nil {
			log.Errorf("MVCCResolveWriteIntent resolvedTxn:%v, Key:%q, error:%s", resolveTxn, resolveKey, errors.ErrorStack(err))
		}
	}

	return nil
}

func mvccResolveIntentPending(engine Engine, key meta.Key, txn *meta.Transaction, localMeta *MVCCMetadata,
	stats *meta.RangeStats, origKeySize int, origValSize int) error {
	if !txn.Timestamp.Large(localMeta.Timestamp) {
		return nil
	}

	if _, _, err := MVCCPutMeta(engine, key, txn, localMeta.Deleted, stats, origKeySize, origValSize); err != nil {
		return errors.Trace(err)
	}

	if err := mvccResolveIntentUpdate(engine, key, txn, localMeta, stats); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func mvccResolveIntentCommit(engine Engine, key meta.Key, txn *meta.Transaction, localMeta *MVCCMetadata,
	stats *meta.RangeStats, origKeySize int, origValSize int) error {
	if _, _, err := MVCCPutMeta(engine, key, txn, localMeta.Deleted, stats, origKeySize, origValSize); err != nil {
		return errors.Trace(err)
	}

	if !txn.Timestamp.Large(localMeta.Timestamp) {
		return nil
	}

	if err := mvccResolveIntentUpdate(engine, key, txn, localMeta, stats); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func mvccResolveIntentUpdate(engine Engine, key meta.Key, txn *meta.Transaction, localMeta *MVCCMetadata,
	stats *meta.RangeStats) error {
	val, mvccKeySize, bytesValSize, err := MVCCGet(engine, key, localMeta.Timestamp)
	if err != nil {
		return errors.Trace(err)
	}

	if err = MVCCPut(engine, key, txn, *val, stats); err != nil {
		return errors.Trace(err)
	}

	if err := MVCCClear(engine, key, localMeta.Timestamp); err != nil {
		return errors.Trace(err)
	}

	stats.UpdateRangeStats(mvccKeySize, bytesValSize, 0, 0)

	return nil
}

func mvccResolveIntentAbort(engine Engine, key meta.Key, txn *meta.Transaction, localMeta *MVCCMetadata,
	stats *meta.RangeStats, origKeySize int, origValSize int) error {
	if err := MVCCClear(engine, key, localMeta.Timestamp); err != nil {
		return errors.Trace(err)
	}

	stats.UpdateRangeStats(origKeySize, origValSize, 0, 0)

	latestKey := meta.NewMVCCTimeKey(key, localMeta.Timestamp)
	// Compute the next possible mvcc value for this key.
	nextKey := latestKey.Next()
	// Compute the last possible mvcc value for this key.
	endScanKey := meta.NewMVCCKey(key.Next())
	kvs, err := Scan(engine, nextKey, endScanKey, 1)
	if err != nil {
		return errors.Trace(err)
	}

	if len(kvs) == 0 {
		//clear meta key, if no data
		if err = engine.Clear(meta.NewMVCCKey(key)); err != nil {
			return errors.Trace(meta.NewEngineClearDataError(key, err.Error()))
		}
		stats.UpdateRangeStats(origKeySize, origValSize, 0, 0)
		return nil
	}

	_, txn.Timestamp, _, err = kvs[0].Key.Decode()
	if err != nil {
		return errors.Trace(err)
	}

	mvccVal := MVCCValue{}
	if err = mvccVal.Unmarshal(kvs[0].Value); err != nil {
		return errors.Trace(meta.NewUnmarshalDataError(kvs[0].Value, err.Error()))
	}

	if _, _, err := MVCCPutMeta(engine, key, txn, len(mvccVal.Value.Bytes) == 0, stats, origKeySize, origValSize); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// MVCCClear clear the mvcc data
func MVCCClear(engine Engine, key meta.Key, timestamp meta.Timestamp) error {
	latestKey := meta.NewMVCCTimeKey(key, timestamp)
	if err := engine.Clear(latestKey); err != nil {
		return errors.Trace(meta.NewEngineClearDataError(key, fmt.Sprintf("timestamp:%v err:%v", timestamp.String(), err)))
	}
	return nil
}

// GetMetaKey get MetaKey from iterator
func GetMetaKey(iter Iterator, encEndKey meta.MVCCKey, reverse bool) (meta.Key, meta.MVCCKey, error) {
	metaKey := iter.Key()
	if metaKey.Less(encEndKey) == reverse {
		return nil, nil, nil
	}

	key, _, isValue, err := metaKey.Decode()
	if err != nil {
		return nil, nil, errors.Annotate(err, fmt.Sprintf("metaKey:%q", metaKey))
	}

	if isValue {
		metaKey = metaKey[:len(metaKey)-meta.EncodeTimestampSize]
	}

	return key, metaKey, nil
}
