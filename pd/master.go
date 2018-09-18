package pd

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/juju/errors"
	"github.com/zssky/log"
	"golang.org/x/net/context"

	"github.com/taorenhai/ancestor/util/retry"
)

func (s *Server) setIsMaster(b bool) {
	value := int64(0)
	if b {
		value = 1
	}

	atomic.StoreInt64(&s.isMaster, value)
}

func (s *Server) checkIsMaster() bool {
	return atomic.LoadInt64(&s.isMaster) == 1
}

func (s *Server) setIsReady(b bool) {
	value := int64(0)
	if b {
		value = 1
	}

	atomic.StoreInt64(&s.isReady, value)
}

func (s *Server) checkIsReady() bool {
	return atomic.LoadInt64(&s.isReady) == 1
}

// getMaster gets server master from etcd.
func (s *Server) getMaster() (string, error) {
	kvs, err := getValues(s.etcdClient, s.masterPath)
	if err != nil {
		return "", errors.Trace(err)
	}

	if kvs == nil {
		return "", nil
	}

	return string(kvs[0].Value), nil
}

func (s *Server) newMasterCmp() clientv3.Cmp {
	return clientv3.Compare(clientv3.Value(s.masterPath), "=", s.masterValue)
}

func (s *Server) masterLoop() {
	s.stopper.RunWorker(func() {
		for r := retry.Start(retry.Options{}); ; r.Next() {
			s.setIsMaster(false)
			s.setIsReady(false)

			master, err := s.getMaster()
			if err != nil {
				log.Errorf("get master err %v", err)
				continue
			}
			if len(master) != 0 && strings.Compare(master, s.masterValue) != 0 {
				log.Debugf("master is %s, watch it", string(master))

				s.watchMaster()

				log.Debug("master changed, try to campaign master")
			}

			if err = s.campaignAndRunMaster(master); err != nil {
				log.Errorf("campaign master err %s", err)
			}
		}
	})
}

func (s *Server) watchMaster() {
	watcher := clientv3.NewWatcher(s.etcdClient)
	defer watcher.Close()

	for {
		watchChan := watcher.Watch(s.etcdClient.Ctx(), s.masterPath)
		for resp := range watchChan {
			if resp.Canceled {
				return
			}
			for _, ev := range resp.Events {
				if ev.Type == mvccpb.DELETE {
					return
				}
			}
		}
	}
}

func (s *Server) campaignAndRunMaster(master string) error {
	log.Debug("begin to campaign master")

	lessor := clientv3.NewLease(s.etcdClient)
	defer lessor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), etcdTimeout)
	leaseResp, err := lessor.Grant(ctx, s.cfg.MasterLease)
	cancel()
	if err != nil {
		return errors.Trace(err)
	}

	var cmp clientv3.Cmp
	if strings.Compare(master, s.masterValue) == 0 {
		cmp = s.newMasterCmp()
	} else {
		cmp = clientv3.Compare(clientv3.CreateRevision(s.masterPath), "=", 0)
	}

	ops := clientv3.OpPut(s.masterPath, s.masterValue, clientv3.WithLease(clientv3.LeaseID(leaseResp.ID)))
	if err = putValues(s.etcdClient, cmp, ops); err != nil {
		return errors.Trace(err)
	}

	log.Debug("campaign master ok")
	s.setIsMaster(true)
	defer s.setIsMaster(false)

	log.Debugf("KeepAlive")
	ch, err := lessor.KeepAlive(s.etcdClient.Ctx(), clientv3.LeaseID(leaseResp.ID))
	if err != nil {
		return errors.Trace(err)
	}

	log.Debug("sync timestamp")
	if err = s.loadTimestamp(); err != nil {
		return errors.Trace(err)
	}

	log.Debug("sync range descripters")
	if err = s.region.loadRangeDescriptors(); err != nil {
		return errors.Trace(err)
	}

	log.Debug("sync node descripters")
	if err = s.cluster.loadNodes(); err != nil {
		return errors.Trace(err)
	}

	s.setIsReady(true)
	defer s.setIsReady(false)

	ticker := time.NewTicker(time.Duration(updateTimestampStep) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				log.Info("keep alive channel is closed")
				return nil
			}
		case <-ticker.C:
			if err = s.saveTimestamp(); err != nil {
				return errors.Trace(err)
			}
		case <-s.stopper.ShouldStop():
			log.Info("master stop")
			return nil
		}
	}
}
