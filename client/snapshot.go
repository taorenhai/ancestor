package client

import (
	"io"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
)

// snapshot implements Snapshot
type snapshot struct {
	*dbTxn
}

// newSnapshot create new snapshot
func newSnapshot(txn *dbTxn) *snapshot {
	return &snapshot{
		dbTxn: txn,
	}
}

// Get retrieves the value for a key, returning the retrieved key/value or an error.
func (s *snapshot) Get(k meta.Key) ([]byte, error) {
	if len(k) == 0 {
		return nil, ErrInvalidKey
	}

	var val []byte

	key := meta.NewKey(util.UserPrefix, k)

	return val, s.doNodeRequest(key, func(node *nodeAPI, rd meta.RangeDescriptor) error {
		breq := newBatchRequest(key, rd.RangeID)
		breq.addGet(&s.dbTxn.Transaction, key)

		resp, err := breq.runBatchWithResult(node)
		if err != nil {
			if _, ok := errors.Cause(err).(*meta.TransactionAbortedError); ok {
				s.Priority++
				log.Debugf("%s inc Priority:%d", s.Name, s.Priority)
				return meta.NewReadPriorityError()
			}
			return errors.Trace(err)
		}

		var kvs []meta.KeyValue

		if err = s.parseBatchGetResponse(resp, &kvs); err != nil {
			return errors.Trace(err)
		}

		if len(kvs[0].Value.Bytes) == 0 {
			return errors.Trace(meta.NewNotExistError(kvs[0].Key))
		}

		val = kvs[0].Value.Bytes
		return nil
	})
}

//Seek return new iter
func (s *snapshot) Seek(k meta.Key) (Iterator, error) {
	log.Infof("%s seek key:%q", s.Name, k)
	if len(k) == 0 {
		return nil, ErrInvalidKey
	}

	if err := s.doPutData(); err != nil {
		return nil, errors.Trace(err)
	}

	iter := newIter(s.dbTxn, k)

	return iter, iter.Next()
}

func (s *snapshot) createBatchRequests(vals map[string][]byte) (map[meta.RangeID]*batchRequest, error) {
	m := make(map[meta.RangeID]*batchRequest)

	for k, v := range vals {
		if len(v) != 0 {
			continue
		}

		key := meta.NewKey(util.UserPrefix, []byte(k))
		rd, err := s.getRangeDescFromCache(key)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if _, ok := m[rd.RangeID]; !ok {
			m[rd.RangeID] = newBatchRequest(key, rd.RangeID)
		}
		m[rd.RangeID].addGet(&s.Transaction, key)
	}

	return m, nil
}

func (s *snapshot) parseBatchGetResponse(rs []meta.Response, kvs *[]meta.KeyValue) error {
	for _, resp := range rs {
		r, ok := resp.(*meta.GetResponse)
		if !ok {
			return errors.Trace(errors.New("response to getResponse error"))
		}
		kv := meta.KeyValue{Key: r.KeyValue.Key[len(util.UserPrefix):], Value: r.KeyValue.Value}
		*kvs = append(*kvs, kv)
	}
	return nil
}

func (s *snapshot) doBatchGet(vals map[string][]byte) ([]meta.KeyValue, error) {
	var kvs []meta.KeyValue

	breqs, err := s.createBatchRequests(vals)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(breqs) == 0 {
		return nil, io.EOF
	}

	for _, breq := range breqs {
		err := s.doNodeRequest(breq.Key, func(node *nodeAPI, rd meta.RangeDescriptor) error {
			rs, err := breq.runBatchWithResult(node)
			if err != nil {
				switch errors.Cause(err).(type) {
				case *meta.TransactionAbortedError:
					s.Priority++
					return meta.NewReadPriorityError()
				case *meta.RangeNotFoundError:
					rangeCache.clear()
					return nil
				}
				return errors.Trace(err)
			}

			if err = s.parseBatchGetResponse(rs, &kvs); err != nil {
				return errors.Trace(err)
			}

			return nil
		})

		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return kvs, nil
}

func (s *snapshot) BatchGet(keys []meta.Key) (map[string][]byte, error) {
	vals := make(map[string][]byte)
	for _, k := range keys {
		vals[string(k)] = nil
	}

	log.Infof("%s snapshotBatchGet", s.Name)

	for kvs, err := s.doBatchGet(vals); err != io.EOF; kvs, err = s.doBatchGet(vals) {
		if err != nil {
			return nil, errors.Trace(err)
		}

		for _, kv := range kvs {
			if len(kv.Value.Bytes) == 0 {
				delete(vals, string(kv.Key))
				continue
			}
			vals[string(kv.Key)] = kv.Value.Bytes
		}
	}

	return vals, nil
}

// Release snapshot release
func (s *snapshot) Release() {
}

// SeekReverse creates a reversed Iterator positioned on the first entry which key is less than k.
// The returned iterator will iterate from greater key to smaller key.
func (s *snapshot) SeekReverse(k meta.Key) (Iterator, error) {
	//TODO add SeekReverse
	return nil, nil
}
