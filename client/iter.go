package client

import (
	"bytes"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
)

type iter struct {
	key     meta.Key
	result  *meta.KeyValue
	scanKey meta.Key
	*dbTxn
}

func newIter(txn *dbTxn, key meta.Key) *iter {
	return &iter{
		key:     meta.NewKey(util.UserPrefix, []byte(key)),
		scanKey: meta.NewKey(util.UserPrefix, []byte(key)),
		dbTxn:   txn,
	}
}

func (it *iter) scanMutilRange(scanKey meta.Key) (*meta.KeyValue, meta.Key, error) {
	var err error
	var kv []meta.KeyValue
	var currRD meta.RangeDescriptor

	for ; len(kv) == 0; scanKey = currRD.EndKey {
		if bytes.HasPrefix(scanKey, util.EndPrefix) {
			return nil, nil, nil
		}
		err = it.doNodeRequest(scanKey, func(node *nodeAPI, rd meta.RangeDescriptor) error {
			currRD = rd
			kv, err = node.scan(&it.dbTxn.Transaction, rd.RangeID, scanKey, rd.EndKey, seekScanSize)
			if err != nil {
				if _, ok := errors.Cause(err).(*meta.TransactionAbortedError); ok {
					it.Priority++
					log.Debugf("%s inc Priority:%d", it.Name, it.Priority)
					return meta.NewReadPriorityError()
				}
			}
			return err
		})
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	}

	if len(kv) == 1 {
		return &kv[0], currRD.EndKey, nil
	}

	return &kv[0], kv[1].Key, nil

}

// Next scan current key
func (it *iter) Next() error {
	var err error
	log.Debugf("%s iter search key:%q, next key:%q", it.Name, it.key, it.scanKey)
	it.result, it.scanKey, err = it.scanMutilRange(it.scanKey)
	if err != nil {
		log.Debugf("%s iter scan key:%q error:%s", it.Name, it.scanKey, errors.ErrorStack(err))
		return nil
	}
	return nil
}

// Value get current result's value
func (it *iter) Value() []byte {
	if it.result != nil {
		log.Debugf("%s iter Value key:%s, result:%q", it.Name, (it.result.Key), it.result.GetValue().Bytes)
		return it.result.GetValue().Bytes
	}
	log.Debugf("%s iter Value key:%s, no found value", it.Name, (it.result.Key))
	return nil
}

// Key get current result's key
func (it *iter) Key() meta.Key {
	if it.result != nil {
		return it.result.Key[len(util.UserPrefix):]
	}
	log.Debugf("iter current key is nil")
	return nil
}

// Valid check result
func (it *iter) Valid() bool {
	if it.result == nil {
		log.Debugf("%s iter valid false", it.Name)
		return false
	}
	log.Debugf("%s iter valid true", it.Name)
	return true
}

// Close set eof true
func (it *iter) Close() {
	log.Debugf("%s iter close", it.Name)
	it.result = nil
}
