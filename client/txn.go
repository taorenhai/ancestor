package client

import (
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
	"github.com/taorenhai/ancestor/util/cache"
	"github.com/taorenhai/ancestor/util/uuid"
)

const (
	seekScanSize    = 2
	maxBatchPutSize = 1000
)

// dbTxn implements Transaction
type dbTxn struct {
	dirty         bool
	hasBegin      bool
	resolveSender *resolveIntent
	kvCache       *cache.OrderedCache
	snapshot      *snapshot
	admin         *admin

	meta.Transaction
	*base
	*heartbeat
}

// newTransaction create new transaction
func newTransaction(b *base, a *admin, f *factory, h *heartbeat, ver *meta.Timestamp, priority int) (*dbTxn, error) {
	timestamp, err := a.GetTimestamp()
	if err != nil {
		log.Errorf("GetTimestamp error:%s", err.Error())
		return nil, errors.Trace(err)
	}

	// snapshot timestamp must be less current timestamp
	if ver != nil {
		if ver.Less(timestamp) {
			timestamp = *ver
		}
	}

	id := uuid.NewUUID4()

	txn := &dbTxn{
		Transaction: meta.Transaction{
			Priority:      int32(priority),
			Name:          id.Short(),
			ID:            meta.NewKey(util.TxnPrefix, []byte(id.String())),
			Status:        meta.PENDING,
			Timestamp:     timestamp,
			LastHeartbeat: timestamp.WallTime,
		},
		resolveSender: newResolveIntent(b, f),
		base:          b,
		admin:         a,
		heartbeat:     h,
		kvCache:       cache.NewOrderedCache(cache.Config{Policy: cache.CacheNone}),
	}

	txn.snapshot = newSnapshot(txn)

	return txn, nil
}

// BeginTransaction begin transaction
func (txn *dbTxn) BeginTransaction() error {
	log.Debugf("%s begin transaction", txn.Name)
	return txn.doNodeRequest(txn.ID, func(node *nodeAPI, rd meta.RangeDescriptor) error {
		return node.beginTransaction(&txn.Transaction, rd.RangeID)
	})
}

// EndTransaction  sync: clear meta
func (txn *dbTxn) EndTransaction(commit bool) error {
	var err error
	//stop txn heartbeat
	txn.del(string(txn.ID))

	if txn.Timestamp, err = txn.admin.GetTimestamp(); err != nil {
		return errors.Trace(err)
	}

	err = txn.doNodeRequest(txn.ID, func(node *nodeAPI, rd meta.RangeDescriptor) error {
		return node.endTransaction(&txn.Transaction, rd.RangeID, commit)
	})

	if _, ok := err.(*meta.NeedRetryError); ok {
		txn.Status = meta.ABORTED
	}

	if err == nil && txn.dirty {
		txn.resolveSender.send(txn)
	}

	return errors.Trace(err)
}

// Get retrieves the value for a key, returning the retrieved key/value or an error.
func (txn *dbTxn) Get(k meta.Key) ([]byte, error) {
	key := meta.NewKey(util.UserPrefix, []byte(k))
	if v, ok := txn.kvCache.Get(key); ok {
		if v == nil {
			return nil, errors.Trace(meta.NewNotExistError(key))
		}
		return v.([]byte), nil
	}
	return txn.snapshot.Get(k)
}

// Seek return new iter
func (txn *dbTxn) Seek(k meta.Key) (Iterator, error) {
	return txn.snapshot.Seek(k)
}

// Set sets the value for key k as v into kvCache.
func (txn *dbTxn) Set(k meta.Key, v []byte) error {
	if len(v) == 0 {
		return errors.Trace(ErrInvalidValue)
	}
	key := meta.NewKey(util.UserPrefix, []byte(k))
	txn.dirty = true
	txn.kvCache.Add(key, v)
	return nil
}

// Delete removes the entry for key k from kvCache.
func (txn *dbTxn) Delete(k meta.Key) error {
	txn.dirty = true
	key := meta.NewKey(util.UserPrefix, []byte(k))
	txn.kvCache.Add(key, nil)
	return nil
}

// Commit commits the transaction operations to KV store.
func (txn *dbTxn) Commit() error {
	log.Debugf("%s commit", txn.Name)
	if !txn.dirty {
		return nil
	}

	if err := txn.doPutData(); err != nil {
		return errors.Trace(err)
	}

	txn.Status = meta.COMMITTED

	return txn.EndTransaction(true)
}

// Rollback undoes the transaction operations to KV store.
func (txn *dbTxn) Rollback() error {
	log.Debugf("%s rollback", txn.Name)
	txn.Status = meta.ABORTED
	if txn.hasBegin {
		return txn.EndTransaction(false)
	}
	return nil
}

// String implements fmt.Stringer interface.
func (txn *dbTxn) String() string {
	log.Debugf("%s string", txn.Name)
	return string(txn.ID)
}

func (txn *dbTxn) createBatchPutRequest(breqs map[meta.RangeID]*batchRequest) error {
	var err error

	txn.kvCache.Walk(func(k, v interface{}) bool {
		var val []byte
		key := k.(meta.Key)

		if v != nil {
			val = v.([]byte)
		}

		rd, e := txn.getRangeDescFromCache(key)
		if e != nil {
			err = e
			return true
		}

		breq, ok := breqs[rd.RangeID]
		if !ok {
			breq = newBatchRequest(key, rd.RangeID)
			breqs[rd.RangeID] = breq
		}

		breq.addPut(&txn.Transaction, key, meta.Value{Bytes: val})

		return false
	})

	return err
}

func (txn *dbTxn) splitBatchPutRequest(breqs map[meta.RangeID]*batchRequest, call func(*batchRequest) error) error {
	for _, breq := range breqs {
		for _, req := range breq.GetReq() {
			txn.resolveSender.add(req.GetInner().Header().RangeID, req.GetInner().Header().Key)
		}

		reqs := breq.Req
		size := len(reqs)

		for s, e := 0, 0; s != size; s = e {
			e = s + maxBatchPutSize
			if e > size {
				e = size
			}
			breq.Req = reqs[s:e]
			if err := call(breq); err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

func (txn *dbTxn) doPutData() error {
	if !txn.hasBegin {
		if err := txn.BeginTransaction(); err != nil {
			return errors.Trace(err)
		}
		txn.hasBegin = true
		txn.add(txn.Transaction)
	}

	if txn.kvCache.Len() == 0 {
		return nil
	}

	breqs := make(map[meta.RangeID]*batchRequest)

	if err := txn.createBatchPutRequest(breqs); err != nil {
		return errors.Trace(err)
	}

	err := txn.splitBatchPutRequest(breqs, func(breq *batchRequest) error {
		err := txn.doNodeRequest(breq.Key, func(node *nodeAPI, rd meta.RangeDescriptor) error {
			_, err := breq.runBatchWithResult(node)
			return err
		})
		if err != nil {
			if _, ok := errors.Cause(err).(*meta.TransactionAbortedError); ok {
				return errors.Trace(meta.NewNeedRetryError(err.Error()))
			}
			return errors.Trace(err)
		}
		return nil
	})
	if err != nil {
		return errors.Trace(err)
	}

	txn.kvCache.Clear()
	return nil
}

// IsReadOnly return true when transaction has write data.
func (txn *dbTxn) IsReadOnly() bool {
	return !txn.dirty
}

// StartTS returns the transaction start timestamp
func (txn *dbTxn) StartTS() uint64 {
	return uint64(txn.Timestamp.WallTime)
}

// SeekReverse creates a reversed Iterator positioned on the first entry which key is less than k.
func (txn *dbTxn) SeekReverse(k meta.Key) (Iterator, error) {
	log.Debugf("%s SeekReverse key:%q", txn.Name, k)
	return txn.snapshot.SeekReverse(k)
}
