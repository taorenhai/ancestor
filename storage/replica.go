package storage

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util/encoding"
)

const (
	// The prescribed length for each command ID.
	raftCommandIDLen = 8

	// for replica storing raft request message with channel buffer
	raftRequestChanSize = 128

	// Ticker every 200 millisecond to notify raft
	raftTickInterval = 200

	raftElectionTick    = 20
	raftHeartbeatTick   = 2
	raftMaxSizePerMsg   = 1024 * 1024
	raftMaxInflightMsgs = 1024

	proposeIdleTimeout = 16 * time.Second
)

type raftAddConf struct {
	nodeID uint64
	done   chan error
}

type raftDeleteConf struct {
	nodeID uint64
	done   chan error
}

// A PendingCmd holds a done channel for a command sent to Raft. Once
// committed to the Raft log, the command is executed and the result returned
// via the done channel.
type pendingCmd struct {
	raftCmd  meta.RaftCommand
	response *meta.BatchResponse
	done     chan meta.ResponseWithError // Used to signal waiting RPC handler
}

// A replica is a contiguous keyspace with writes managed via an instance of the Raft
// consensus algorithm.
type replica struct {
	rangeID meta.RangeID // Should only be set by the constructor.
	store   *store

	rangeDesc          meta.RangeDescriptor
	replicaID          meta.ReplicaID
	raftWakeChan       chan struct{}
	raftStopChan       chan struct{}
	pendingCmds        map[string]*pendingCmd
	pendingProposeChan chan *pendingCmd
	raftAddConfChan    chan *raftAddConf
	raftDeleteConfChan chan *raftDeleteConf
	raftRequestChan    chan *meta.RaftMessageRequest
	raftSnapshotChan   chan raftpb.Snapshot
	raftGroup          *raft.RawNode
	raftOffsetIndex    uint64
	raftAppliedIndex   uint64 // Last index applied to the state machine.
	raftTruncateIndex  uint64
	raftTruncateTerm   uint64
	raftLastIndex      uint64 // Last index persisted to the raft log.
}

// newReplica initializes the replica using the give data, create raft via
// the rawnode interface.
func newReplica(rd *meta.RangeDescriptor, store *store, needLoad bool) (*replica, error) {
	r := &replica{
		rangeID:            rd.RangeID,
		store:              store,
		rangeDesc:          *rd,
		pendingCmds:        make(map[string]*pendingCmd),
		pendingProposeChan: make(chan *pendingCmd, raftRequestChanSize),
		raftAddConfChan:    make(chan *raftAddConf),
		raftDeleteConfChan: make(chan *raftDeleteConf),
		raftRequestChan:    make(chan *meta.RaftMessageRequest, raftRequestChanSize),
		raftSnapshotChan:   make(chan raftpb.Snapshot),
		raftWakeChan:       make(chan struct{}),
		raftStopChan:       make(chan struct{}),
	}

	if needLoad {
		r.loadRaftPrevIndex()
	}

	if err := r.addRaftGroup(rd, store); err != nil {
		return nil, err
	}

	r.processRaft()

	return r, nil
}

func (r *replica) loadRaftPrevIndex() {
	r.raftAppliedIndex, _ = r.loadAppliedIndex()
	r.raftOffsetIndex, _ = r.loadOffsetIndex()
	r.raftLastIndex, _ = r.loadLastIndex()
}

// create new rawnode to replica via raft rawnode interface
func (r *replica) addRaftGroup(rd *meta.RangeDescriptor, store *store) error {
	var peers []raft.Peer
	for _, rep := range rd.Replicas {
		peers = append(peers, raft.Peer{ID: uint64(rep.NodeID) + uint64(r.rangeID)})
	}

	c := &raft.Config{
		ID:              uint64(store.nodeID) + uint64(r.rangeID),
		ElectionTick:    raftElectionTick,
		HeartbeatTick:   raftHeartbeatTick,
		MaxSizePerMsg:   raftMaxSizePerMsg,
		MaxInflightMsgs: raftMaxInflightMsgs,
		Applied:         r.raftAppliedIndex,
		Storage:         r,
	}

	rn, err := raft.NewRawNode(c, peers)
	if err != nil {
		return fmt.Errorf("range:%v create raw node failed, %v", rd.RangeID, err)
	}
	r.raftGroup = rn

	return nil
}

// Send adds a command for execution on this range. The command's
// affected keys are verified to be contained within the range and the
// range's leadership is confirmed. The command is then dispatched
// either along the read-only execution path or the read-write Raft
// command queue.
func (r *replica) Send(breq *meta.BatchRequest) (*meta.BatchResponse, error) {
	// Differentiate between leader, read-only and write.
	if breq.IsLeader() {
		return breq.Execute(
			func(req meta.Request) (meta.Response, error) {
				return r.executeCmd(req, nil)
			})
	}
	if breq.IsReadOnly() || breq.IsWrite() {
		select {
		case respWithErr := <-r.proposeRaftCommand(breq):
			return respWithErr.Reply, respWithErr.Err
		case <-time.After(proposeIdleTimeout):
			return &meta.BatchResponse{}, errors.Errorf("propose request timeout")
		}
	}

	return &meta.BatchResponse{}, errors.Errorf("don't know how to handle command %v", breq.String())
}

// proposeRaftCommand prepares necessary pending command struct and
// initializes a client command ID if one hasn't been. It then
// proposes the command to Raft and returns the error channel and
// pending command struct for receiving.
func (r *replica) proposeRaftCommand(breq *meta.BatchRequest) <-chan meta.ResponseWithError {
	cmd := &pendingCmd{
		raftCmd: meta.RaftCommand{
			RangeID: r.rangeID,
			Request: *breq,
		},
		done: make(chan meta.ResponseWithError, 1),
	}
	r.pendingProposeChan <- cmd

	return cmd.done
}

func (r *replica) preprocessWriteIntentError(breq *meta.BatchRequest, err error) bool {
	if err == nil {
		return false
	}

	wiErr, ok := errors.Cause(err).(*meta.WriteIntentError)
	if !ok {
		return false
	}

	for _, pusheeTxn := range breq.GetPusheeTxn() {
		if bytes.Equal(wiErr.Intent.Txn.ID, pusheeTxn.ID) {
			intents := []meta.Intent{{Span: wiErr.Intent.Span, Txn: pusheeTxn}}
			if err = engine.MVCCResolveWriteIntent(r.store.engine, intents, r.store.rangeStats[r.rangeID]); err != nil {
				log.Errorf("resolveIntent %v error:%s", intents, errors.ErrorStack(err))
				return false
			}
			log.Debugf("[%d %d] ResolveIntent key:%q, pusheeTxn:%s", r.store.nodeID, r.rangeID, wiErr.Intent.Key, pusheeTxn.Name)
			return true
		}
	}

	return false
}

// processRaftCommand processes a raft command by unpacking the command struct to get args and
// reply and then applying the command to the state machine. The error result is sent on the
// command's done channel, if available.
func (r *replica) processRaftCommand(data *meta.RaftData, index uint64) {
	defer r.setAppliedIndex(r.store.engine, index)

	cmd, ok := r.pendingCmds[data.Id]
	if ok {
		delete(r.pendingCmds, data.Id)
	}

	switch data.Cmd.Request.Flags() {
	case meta.IsRead:
		if ok {
			r.store.asyncRequestChan <- &asyncRequest{replica: r, cmd: cmd, snapshot: r.store.engine.NewSnapshot()}
		}
	case meta.IsWrite:
		bresp, err := data.Cmd.Request.Execute(
			func(req meta.Request) (meta.Response, error) {
				resp, err := r.executeCmd(req, nil)
				if r.preprocessWriteIntentError(&data.Cmd.Request, err) {
					return r.executeCmd(req, nil)
				}
				return resp, err
			})
		if ok {
			cmd.done <- meta.ResponseWithError{Err: err, Reply: bresp}
		}
	default:
		log.Warning("[%d %v] processRaftCommand not found request type", r.store, r.rangeID)
	}
}

func (r *replica) processRaft() {
	r.store.stopper.RunWorker(func() {
		ticker := time.Tick(raftTickInterval * time.Millisecond)
		for {
			if r.raftGroup.HasReady() {
				rd := r.raftGroup.Ready()
				if err := r.handleRaftReady(rd); err != nil {
					log.Errorf("range:%v handle raft ready, error:%v", r.rangeID, err)
				}
			}

			select {
			case <-ticker:
				r.raftGroup.Tick()
			case req := <-r.raftRequestChan:
				if err := r.handleRaftMessage(req); err != nil {
					log.Errorf("range:%v handle raft message, error:%v", r.rangeID, err)
				}
			case <-r.raftWakeChan:
			case <-r.store.stopper.ShouldStop():
				return
			case <-r.raftStopChan:
				return
			case cmd := <-r.pendingProposeChan:
				if err := r.handleProposeCmd(cmd); err != nil {
					log.Errorf("range:%v propose cmd:%v, error:%v", r.rangeID, cmd, err)
				}
			case op := <-r.raftAddConfChan:
				r.handleRaftAddConf(op)
			case op := <-r.raftDeleteConfChan:
				r.handleRaftDeleteConf(op)
			}
		}
	})
}

func (r *replica) handleRaftAddConf(op *raftAddConf) {
	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  op.nodeID,
		Context: []byte(fmt.Sprintf("%d", op.nodeID)),
	}

	op.done <- r.raftGroup.ProposeConfChange(cc)
}

func (r *replica) handleRaftDeleteConf(op *raftDeleteConf) {
	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeRemoveNode,
		NodeID:  op.nodeID,
		Context: []byte(fmt.Sprintf("%d", op.nodeID)),
	}

	op.done <- r.raftGroup.ProposeConfChange(cc)
}

func (r *replica) handleProposeCmd(cmd *pendingCmd) error {
	buf := make([]byte, 0, raftCommandIDLen)
	buf = encoding.EncodeUint64(buf, uint64(rand.Int63()))
	idKey := string(buf)

	if _, ok := r.pendingCmds[idKey]; ok {
		return errors.Errorf("range:%v cmd already exists %s", r.rangeID, idKey)
	}

	raftData := &meta.RaftData{Id: idKey, Cmd: cmd.raftCmd}
	data, err := raftData.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	r.pendingCmds[idKey] = cmd
	if err := r.raftGroup.Propose(data); err != nil {
		log.Errorf("range:%v propose data to raft error:%v", r.rangeID, err)
		return err
	}

	r.enqueueRaftUpdateCheck()

	return nil
}

// handleRaftMessage handles message from replica's channel, mainly it's peers's raft message
func (r *replica) handleRaftMessage(req *meta.RaftMessageRequest) error {
	if r.rangeID != req.RangeID {
		return errors.Errorf("range:%v receive message range:%v error", r.rangeID, req.RangeID)
	}

	if req.ToNodeID != r.store.nodeID {
		return errors.Errorf("range:%v node:%v should not receive node:%v message",
			r.rangeID, r.store.nodeID, req.ToNodeID)
	}

	if err := r.raftGroup.Step(req.Message); err != nil {
		return errors.Errorf("range:%v step Message(%v), error:%v", r.rangeID,
			raft.DescribeMessage(req.Message, nil), err)
	}
	r.enqueueRaftUpdateCheck()

	return nil
}

func (r *replica) handleRaftReady(rd raft.Ready) error {
	var err error

	if !raft.IsEmptySnap(rd.Snapshot) {
		if err = r.applySnapshot(rd.Snapshot); err != nil {
			return errors.Trace(err)
		}
	}

	if len(rd.Entries) > 0 {
		if err = r.append(rd.Entries); err != nil {
			return errors.Trace(err)
		}
	}

	if !raft.IsEmptyHardState(rd.HardState) {
		if err = r.setHardState(&rd.HardState); err != nil {
			return errors.Trace(err)
		}
	}

	if err := r.store.engine.Commit(); err != nil {
		return err
	}

	for _, m := range rd.Messages {
		r.sendRaftMessage(m)
	}

	for _, e := range rd.CommittedEntries {
		switch e.Type {
		case raftpb.EntryNormal:
			if len(e.Data) == 0 {
				continue
			}
			data := &meta.RaftData{}
			if err := data.Unmarshal(e.Data); err != nil {
				log.Errorf("replica:%v Unmarshal Data, error:%v", r.rangeID, err)
				continue
			}
			r.processRaftCommand(data, e.Index)

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			if err := cc.Unmarshal(e.Data); err != nil {
				return err
			}

			r.raftGroup.ApplyConfChange(cc)
			r.setAppliedIndex(r.store.engine, e.Index)
		}
	}

	r.raftGroup.Advance(rd)

	return nil
}

func (r *replica) sendRaftMessage(m raftpb.Message) {
	if m.To <= uint64(r.rangeID) {
		log.Warningf("range:%v sendRaftMessage m.To:%v below zero",
			r.rangeID, m.To)
		return
	}

	id := meta.NodeID(m.To - uint64(r.rangeID))
	if !r.store.transport.isExistNodeCache(id) {
		nodes, err := r.store.client.GetNodes()
		if err != nil {
			log.Errorf("replia:%v get node list, error:%v", r.rangeID, err)
			return
		}
		r.store.transport.clearNodesCache()
		r.store.transport.updateNodesCache(nodes)

		if !r.store.transport.isExistNodeCache(id) {
			log.Warningf("range:%v not exist node:%v cache", r.rangeID, id)
			return
		}
	}

	err := r.store.transport.send(&meta.RaftMessageRequest{
		RangeID:    r.rangeID,
		FromNodeID: r.store.nodeID,
		ToNodeID:   id,
		Message:    m,
	})

	status := raft.SnapshotFinish
	if err != nil {
		log.Warningf("range:%v on node:%v failed to send message to node:%v, error:%v",
			r.rangeID, r.store.nodeID, id, err)
		r.raftGroup.ReportUnreachable(m.To)
		status = raft.SnapshotFailure
	}

	if m.Type == raftpb.MsgSnap {
		r.raftGroup.ReportSnapshot(m.To, status)
	}
}

// enqueueRaftUpdateCheck checks for raft updates when the processRaft goroutine is idle.
func (r *replica) enqueueRaftUpdateCheck() {
	select {
	case r.raftWakeChan <- struct{}{}:
	default:
	}
}

// getRaftGroupLeader return raft group leader node ID, if not found, will return zero.
func (r *replica) getRaftGroupLeader() (meta.NodeID, bool) {
	leader := atomic.LoadUint64(&r.raftGroup.Status().Lead)
	isLeader := false

	if leader > 0 {
		id := meta.NodeID(leader - uint64(r.rangeID))
		if id == r.store.nodeID {
			isLeader = true
		}

		return id, isLeader
	}

	return meta.NodeID(0), isLeader
}

func (r *replica) validateKeyRange(key meta.Key) bool {
	skey := r.rangeDesc.StartKey
	ekey := r.rangeDesc.EndKey

	if len(ekey) == 0 && bytes.Compare(skey, key) <= 0 {
		return true
	}

	if bytes.Compare(skey, key) <= 0 && bytes.Compare(key, ekey) < 0 {
		return true
	}

	return false
}
