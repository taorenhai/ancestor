package pd

import (
	"encoding/binary"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/juju/errors"
	"github.com/zssky/log"
	"golang.org/x/net/context"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/retry"
)

const (
	requestTimeout = 10 * time.Second
)

var (
	defaultRetryOptions = retry.Options{MaxRetries: 20}
	errSendFail         = errors.New("send batch request failed")
	errLostMaster       = errors.New("maybe we lost master")
)

func getValues(c *clientv3.Client, key string, opts ...clientv3.OpOption) ([]*mvccpb.KeyValue, error) {
	kv := clientv3.NewKV(c)

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	resp, err := kv.Get(ctx, key, opts...)
	cancel()

	if err != nil {
		return nil, errors.Trace(err)
	}

	if n := len(resp.Kvs); n == 0 {
		return nil, nil
	}

	return resp.Kvs, nil
}

func putValues(c *clientv3.Client, cmp clientv3.Cmp, ops ...clientv3.Op) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	resp, err := c.Txn(ctx).
		If(cmp).
		Then(ops...).
		Commit()
	cancel()
	if err != nil {
		return errors.Trace(err)
	}
	if !resp.Succeeded {
		return errors.Trace(errLostMaster)
	}

	return nil
}

func bytesToUint64(b []byte) (uint64, error) {
	if len(b) != 8 {
		return 0, errors.Errorf("invalid data, must 8 bytes, but %d", len(b))
	}

	return binary.BigEndian.Uint64(b), nil
}

func uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func (s *Server) sendBatchRequestToLeader(breq *meta.BatchRequest, bresp *meta.BatchResponse, rd *meta.RangeDescriptor) error {
	id := rd.Replicas[0].NodeID

	for r := retry.Start(defaultRetryOptions); r.Next(); {
		//FIXME can't read ndoes.
		cli, err := s.getNodeClient(s.cluster.nodes[id].Address)
		if err != nil {
			log.Errorf("send request to %d failed: %s", id, err.Error())
			return errors.Trace(errSendFail)
		}

		br, err := cli.NodeMessage(context.TODO(), breq)
		if err != nil {
			//FIXME can't read ndoes.
			s.closeNodeClient(s.cluster.nodes[id].Address)
			id = rd.GetNextNodeID(id)
			log.Errorf("send request to %d failed: %s", id, err.Error())
			continue
		}
		*bresp = *br

		re := bresp.GetError()
		if re == nil {
			s.region.ranges.AdjustLeader(rd.RangeID, id)
			return nil
		}

		switch e := errors.Cause(re.GoError()).(type) {
		case *meta.RangeNotFoundError:
			id = rd.GetNextNodeID(id)
			log.Debugf("%s RangeNotFoundError change next node:%d", s.cluster.nodes[id].Address, id)

		case *meta.NotLeaderError:
			id = e.NodeID
			if id <= 0 {
				id = rd.GetNextNodeID(id)
			}
			log.Debugf("NotLeaderError next leader:%d", id)

		default:
			return errors.Trace(e)
		}
	}

	return errors.Trace(errSendFail)
}

func (s *Server) getNodeClient(host string) (*meta.NodeClient, error) {
	s.Lock()
	defer s.Unlock()

	if c, ok := s.nodeClientCache[host]; ok {
		return c, nil
	}

	c, err := meta.NewNodeClient(host)
	if err != nil {
		log.Errorf("NewNodeClient to %s failed: %s", host, err.Error())
		return nil, err
	}

	s.nodeClientCache[host] = c
	return c, nil
}

func (s *Server) closeNodeClient(host string) {
	s.Lock()
	defer s.Unlock()

	if c, ok := s.nodeClientCache[host]; ok {
		c.Close()
		delete(s.nodeClientCache, host)
	}

}

func (s *Server) sendBatchRequest(breq *meta.BatchRequest, bresp *meta.BatchResponse, id meta.NodeID) error {
	for r := retry.Start(defaultRetryOptions); r.Next(); {
		cli, err := s.getNodeClient(s.cluster.nodes[id].Address)
		if err != nil {
			log.Errorf("send request to %d failed: %s", id, err.Error())
			continue
		}

		br, err := cli.NodeMessage(context.TODO(), breq)
		if err != nil {
			s.closeNodeClient(s.cluster.nodes[id].Address)
			log.Errorf("send request to %d failed: %s", id, err.Error())
			continue
		}
		*bresp = *br

		if pError := bresp.GetError(); pError != nil {
			err := pError.GoError()
			log.Errorf("%s", err.Error())
			continue
		}

		return nil
	}

	return errSendFail
}
