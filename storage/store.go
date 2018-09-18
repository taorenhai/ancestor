package storage

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/taorenhai/ancestor/client"
	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util/cache"
	"github.com/taorenhai/ancestor/util/retry"
	"github.com/taorenhai/ancestor/util/stop"
	"github.com/juju/errors"
	"github.com/zssky/log"
	"google.golang.org/grpc"
)

const (
	asyncChanSize    = 128
	asyncWorkerCount = 32
)

var (
	// defaultRangeRetryOptions are default retry options for retrying commands
	// sent to the store's ranges, for WriteTooOld and WriteIntent errors.
	defaultRangeRetryOptions = retry.Options{
		MaxBackoff: time.Second,
		Multiplier: 2,
	}
)

type asyncRequest struct {
	snapshot engine.Engine
	cmd      *pendingCmd
	replica  *replica
}

// Store represents a node in a cluster.
type Store interface {
	Send(*meta.BatchRequest) (*meta.BatchResponse, error)

	// Create add new replica to store, and one replica contains one rawnode.
	Create([]meta.RangeDescriptor) error
}

// A Store maintains a map of ranges by start key. A Store corresponds to
// one physical device.
type store struct {
	nodeID meta.NodeID
	engine engine.Engine
	sync.Mutex

	transport        *transport
	client           client.Admin
	replicas         map[meta.RangeID]*replica
	stopper          *stop.Stopper
	asyncRequestChan chan *asyncRequest
	rangeStats       map[meta.RangeID]*meta.RangeStats

	activeRaftSnapshot int32
}

// NewStore returns a new instance of a store.
func NewStore(id meta.NodeID, serv *grpc.Server, client client.Admin,
	eng engine.Engine, stopper *stop.Stopper, nc *cache.NodesCache) (Store, error) {
	t := newTransport(serv, stopper, nc)
	s := &store{
		nodeID:           id,
		engine:           eng,
		transport:        t,
		client:           client,
		stopper:          stopper,
		replicas:         make(map[meta.RangeID]*replica),
		asyncRequestChan: make(chan *asyncRequest, asyncChanSize),
	}

	var err error
	if s.rangeStats, err = s.loadRangeStats(); err != nil {
		return nil, errors.Errorf("get local node stats err:%v", err)
	}

	s.transport.listen(s.raftMessage)
	s.uploadRangeStats()
	s.newAsyncRequestWorker()

	log.Debugf("nodeID:%d", id)

	return s, nil
}

func (s *store) raftMessage(batchReq *meta.RaftMessageBatchRequest) {
	for i, req := range batchReq.Requests {
		r, err := s.getReplica(req.RangeID)
		if err != nil {
			log.Warningf("range:%v, %s", req.RangeID, errors.ErrorStack(err))
			continue
		}
		r.raftRequestChan <- &batchReq.Requests[i]
	}
}

// Create create new replicas from RangeDescriptors.
func (s *store) Create(rds []meta.RangeDescriptor) error {
	for _, rd := range rds {
		for _, r := range rd.Replicas {
			if r.NodeID == s.nodeID && !rd.IsSystem() {
				if _, err := s.createReplica(rd, true); err != nil {
					log.Errorf("create new replica:%v, err:%v", rd.RangeID, err)
				}
			}
		}
	}
	return nil
}

// The value of 'needLoad' will be set true, which mainly for creating raft with previous index.
// There are two case setting the value 'needLoad' true:
// one is creating replica from starting state.
// another is Adding a new replica to exist replica's group.
func (s *store) createReplica(rd meta.RangeDescriptor, needLoad bool) (*replica, error) {
	_, err := s.getReplica(rd.RangeID)
	if err == nil {
		return nil, fmt.Errorf("create replica, range:%v already exist", rd.RangeID)
	}

	r, err := newReplica(&rd, s, needLoad)
	if err != nil {
		return nil, errors.Trace(err)
	}

	s.addRangeStats(rd.RangeID)
	s.addReplica(rd.RangeID, r)

	return r, nil
}

func (s *store) addRangeStats(id meta.RangeID) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.rangeStats[id]; ok {
		s.rangeStats[id].RangeStatsInfo.IsRemoved = false
		return
	}
	rs := &meta.RangeStats{
		RangeStatsInfo: &meta.RangeStatsInfo{
			RangeID: id,
			NodeID:  s.nodeID,
		},
	}
	s.rangeStats[id] = rs
}

func (s *store) addReplica(id meta.RangeID, r *replica) {
	s.Lock()
	s.replicas[id] = r
	s.Unlock()
}

func (s *store) getReplica(id meta.RangeID) (*replica, error) {
	s.Lock()
	defer s.Unlock()
	if r, ok := s.replicas[id]; ok {
		return r, nil
	}
	return nil, errors.Trace(meta.NewRangeNotFoundError(id))
}

func (s *store) deleteReplica(id meta.RangeID) error {
	if _, err := s.getReplica(id); err != nil {
		return errors.Trace(err)
	}
	s.Lock()
	delete(s.replicas, id)
	s.Unlock()
	return nil
}

// Send send request to replica from node server.
func (s *store) Send(breq *meta.BatchRequest) (*meta.BatchResponse, error) {
	bresp := &meta.BatchResponse{}
	if breq.IsSystem() {
		return breq.Execute(s.executeRootCmd)
	}

	// get replica, if not exist return err, client clean cache.
	r, err := s.getReplica(breq.RangeID)
	if err != nil {
		log.Errorf("getReplica id:%d error:%s", breq.RangeID, err.Error())
		return bresp, errors.Trace(err)
	}

	// if replica not the request raft group leader, return error with leader id
	id, isLeader := r.getRaftGroupLeader()
	if !isLeader {
		return bresp, errors.Trace(meta.NewNotLeaderError(id))
	}

	// Add the command to the range for execution; exit retry loop on success.
	for rtry := retry.Start(defaultRangeRetryOptions); rtry.Next(); {
		log.Debugf("Send rangeID:%v retryCount:%v, batchCount:%d", breq.RangeID, rtry.CurrentAttempt(), len(breq.Req))

		resp, err := r.Send(breq)
		bresp.Resp = append(bresp.Resp, resp.Resp...)
		if err == nil {
			break
		}

		// jump success request
		breq.Req = breq.Req[len(resp.Resp):]

		//FIXME: remove unused log, figure out a whole error handler mechanism which is cleaner
		if wiErr, ok := errors.Cause(err).(*meta.WriteIntentError); ok {
			log.Debugf("writeIntentError wiErr:%s", wiErr.String())

			pusheeTxn, err1 := s.processWriteIntentError(wiErr)
			if err1 != nil {
				log.Infof("processWriteIntentError %s", errors.ErrorStack(err1))
				return bresp, errors.Trace(err1)
			}
			breq.PusheeTxn = append(breq.PusheeTxn, *pusheeTxn)
			log.Debugf("append batchReq PusheeTxn %v", pusheeTxn)
			continue
		}

		switch errors.Cause(err).(type) {
		case *meta.WriteTooOldError,
			*meta.TransactionStatusError,
			*meta.HeartbeatTransactionError:
			log.Infof("send error:%s", err.Error())
		default:
			log.Errorf("err:%s type:%s", errors.ErrorStack(err), reflect.TypeOf(err).String())
		}
		return bresp, errors.Trace(err)
	}
	return bresp, nil
}

func (s *store) processWriteIntentError(wiErr *meta.WriteIntentError) (*meta.Transaction, error) {
	pusheeTxn, err := s.client.PushTxn(wiErr.PushType, wiErr.Intent.GetTxn().ID, wiErr.GetPusherTxn())
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := s.client.ResolveIntent(wiErr.Intent.Key, pusheeTxn); err != nil {
		return nil, errors.Trace(err)
	}

	return pusheeTxn, nil
}

func (s *store) executeRootCmd(request meta.Request) (meta.Response, error) {
	switch req := request.(type) {
	case *meta.CreateReplicaRequest:
		resp := &meta.CreateReplicaResponse{}
		return resp, s.executeCreateReplica(req)
	case *meta.RemoveReplicaRequest:
		resp := &meta.RemoveReplicaResponse{}
		return resp, s.executeRemoveReplica(req)
	}

	return nil, errors.Errorf("no found request type:%v", reflect.TypeOf(request).String())
}

func (s *store) executeCreateReplica(req *meta.CreateReplicaRequest) error {
	if _, ok := s.replicas[req.RangeID]; ok {
		log.Infof("range:%v already exist", req.RangeID)
		return nil
	}

	if _, err := s.createReplica(req.RangeDescriptor, true); err != nil {
		log.Errorf("createReplica error:%s", err.Error())
		return err
	}

	return nil
}

func (s *store) executeRemoveReplica(req *meta.RemoveReplicaRequest) error {
	r, ok := s.replicas[req.RangeID]
	if !ok {
		return errors.Errorf("range:%v not found", req.RangeID)
	}
	if !isExistNode(req.NodeID, r) {
		return errNotFoundNode
	}

	if err := s.deleteReplica(req.RangeID); err != nil {
		return err
	}
	r.raftStopChan <- struct{}{}
	s.rangeStats[req.RangeID].RangeStatsInfo.IsRemoved = true

	return nil
}

func (s *store) newAsyncRequestWorker() {
	for i := 0; i < asyncWorkerCount; i++ {
		s.stopper.RunWorker(func() {
			s.handleAsyncRequest()
		})
	}
}

func (s *store) handleAsyncRequest() {
	for {
		select {
		case ar := <-s.asyncRequestChan:
			bresp, err := ar.cmd.raftCmd.Request.Execute(
				func(req meta.Request) (meta.Response, error) {
					return ar.replica.executeCmd(req, ar.snapshot)
				})

			ar.cmd.done <- meta.ResponseWithError{Err: err, Reply: bresp}
		case ss := <-s.transport.snapStatusChan:
			if r, err := s.getReplica(ss.req.RangeID); err == nil {
				status := raft.SnapshotFinish
				if ss.err != nil {
					status = raft.SnapshotFailure
				}
				r.raftGroup.ReportSnapshot(ss.req.Message.To, status)
			}
		}
	}
}

// acquireRaftSnapshot returns true if a new raft snapshot can start.
// If true is returned, the caller must call releaseRaftSnapshot.
func (s *store) acquireRaftSnapshot() bool {
	return atomic.CompareAndSwapInt32(&s.activeRaftSnapshot, 0, 1)
}

// releaseRaftSnapshot decrements the count of active snapshots.
func (s *store) releaseRaftSnapshot() {
	atomic.SwapInt32(&s.activeRaftSnapshot, 0)
}
