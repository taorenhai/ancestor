package pd

import (
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

const (
	maxRetryNum = 100
	// update timestamp every updateTimestampStep milliseconds.
	updateTimestampStep = int64(50)
	maxLogical          = int32(1 << 18)
)

type atomicObject struct {
	wallTime time.Time
	logical  int32
}

func (s *Server) loadTimestampFromDB() (int64, error) {
	kvs, err := getValues(s.etcdClient, s.timestampPath)
	if err != nil {
		return 0, errors.Trace(err)
	}
	if kvs == nil {
		return 0, nil
	}

	ts, err := bytesToUint64(kvs[0].Value)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return int64(ts), nil
}

// save timestamp, if lastTs is 0, we think the timestamp doesn't exist, so create it,
// otherwise, update it.
func (s *Server) saveTimestampToDB(now time.Time) error {
	data := uint64ToBytes(uint64(now.UnixNano()))

	ops := clientv3.OpPut(s.timestampPath, string(data))
	if err := putValues(s.etcdClient, s.newMasterCmp(), ops); err != nil {
		return errors.Trace(err)
	}

	s.lastSavedTime = now

	return nil
}

func (s *Server) loadTimestamp() error {
	lastSavedTime, err := s.loadTimestampFromDB()

	if err != nil {
		return errors.Trace(err)
	}

	var nowTime time.Time

	for {
		nowTime = time.Now()

		elapsed := (nowTime.UnixNano() - lastSavedTime) / 1e6
		if elapsed <= 0 {
			return errors.Errorf("%s <= last saved time %s", nowTime, time.Unix(0, lastSavedTime))
		}

		// TODO: can we speed up this?
		if waitTime := 2*s.cfg.TsSaveInterval - elapsed; waitTime > 0 {
			log.Warnf("wait %d milliseconds to guarantee valid generated timestamp", waitTime)
			time.Sleep(time.Duration(waitTime) * time.Millisecond)
			continue
		}

		break
	}

	if err = s.saveTimestampToDB(nowTime); err != nil {
		return errors.Trace(err)
	}

	log.Debug("sync and save timestamp ok")

	cur := &atomicObject{
		wallTime: nowTime,
	}
	s.ts.Store(cur)

	return nil
}

func (s *Server) saveTimestamp() error {
	prev := s.ts.Load().(*atomicObject)
	nowTime := time.Now()

	// ms
	elapsed := nowTime.Sub(prev.wallTime).Nanoseconds() / 1e6
	if elapsed > 3*updateTimestampStep {
		log.Warnf("clock offset: %v, prev: %v, now %v", elapsed, prev.wallTime, nowTime)
	}
	// Avoid the same physical time stamp
	if elapsed <= 0 {
		log.Warningf("invalid physical timestamp, prev:%v, now:%v, re-update later", prev.wallTime, nowTime)
		return nil
	}

	if nowTime.Sub(s.lastSavedTime).Nanoseconds()/1e6 > s.cfg.TsSaveInterval {
		if err := s.saveTimestampToDB(nowTime); err != nil {
			return errors.Trace(err)
		}
	}

	cur := &atomicObject{
		wallTime: nowTime,
	}
	s.ts.Store(cur)

	return nil
}

func (s *Server) getTimestamp() (meta.Timestamp, error) {
	ts := meta.Timestamp{}

	for i := 0; i < maxRetryNum; i++ {
		cur, ok := s.ts.Load().(*atomicObject)
		if !ok {
			log.Errorf("we haven't synced timestamp ok, wait and retry")
			time.Sleep(200 * time.Millisecond)
			continue
		}

		ts.WallTime = int64(cur.wallTime.UnixNano()) / 1e6
		ts.Logical = atomic.AddInt32(&cur.logical, 1)
		if ts.Logical >= maxLogical {
			log.Errorf("logical part outside of max logical interval %v, please check ntp time", ts)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		return ts, nil
	}

	return ts, errors.New("get timestamp failed, please try again")
}
