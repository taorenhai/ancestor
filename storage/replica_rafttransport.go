package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/zssky/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/cache"
	"github.com/taorenhai/ancestor/util/stop"
)

const (
	// Outgoing messages are queued on a per-node basis on a channel of
	// this size.
	raftSendBufferSize = 500

	// When no message has been sent to a Node for that duration, the
	// corresponding instance of processQueue will shut down.
	raftIdleTimeout = 2 * time.Minute

	raftMaxBatchRequestSize = 128
)

type raftMessageHandler func(*meta.RaftMessageBatchRequest)

type raftSnapStatus struct {
	req *meta.RaftMessageRequest
	err error
}

// transport handles the rpc messages for raft group.
type transport struct {
	nodesCache *cache.NodesCache
	rpcServer  *grpc.Server
	stopper    *stop.Stopper
	sync.Mutex
	handler   raftMessageHandler
	queues    map[meta.NodeID]chan *meta.RaftMessageRequest
	grpcCache map[string]*grpc.ClientConn

	snapStatusChan chan *raftSnapStatus
}

// newTransport creates a new transport with specified rpc server.
func newTransport(serv *grpc.Server, stopper *stop.Stopper, nc *cache.NodesCache) *transport {
	t := &transport{
		nodesCache:     nc,
		rpcServer:      serv,
		stopper:        stopper,
		queues:         make(map[meta.NodeID]chan *meta.RaftMessageRequest),
		grpcCache:      make(map[string]*grpc.ClientConn),
		snapStatusChan: make(chan *raftSnapStatus),
	}

	meta.RegisterRaftMessageServer(serv, t)

	return t
}

func (t *transport) isExistNodeCache(id meta.NodeID) bool {
	return t.nodesCache.IsExistNodeCache(id)
}

func (t *transport) updateNodesCache(nds []meta.NodeDescriptor) {
	t.nodesCache.AddNodes(nds)
}

func (t *transport) clearNodesCache() {
	t.nodesCache.ClearNodes()
}

// raftMessage proxies the incoming request to the listening server interface.
func (t *transport) RaftMessage(stream meta.RaftMessage_RaftMessageServer) error {
	ch := make(chan error, 1)

	t.stopper.RunWorker(func() {
		ch <- func() error {
			for {
				req, err := stream.Recv()
				if err != nil {
					return err
				}

				t.handler(req)
			}
		}()
	})

	select {
	case err := <-ch:
		return err
	case <-t.stopper.ShouldStop():
		return stream.SendAndClose(new(meta.RaftMessageBatchResponse))
	}
}

// listen implements the store.transport interface by registering a serverInterface
// to receive proxied messages.
func (t *transport) listen(handler raftMessageHandler) {
	t.handler = handler
}

// processQueue creates a client and sends messages from its designated queue
// via that client, exiting when the client fails or when it idles out. All
// messages remaining in the queue at that point are lost and a new instance of
// processQueue should be started by the next message to be sent.
func (t *transport) processQueue(id meta.NodeID) {
	t.Lock()
	ch, ok := t.queues[id]
	t.Unlock()
	if !ok {
		return
	}
	// Clean-up when the loop below shuts down.
	defer func() {
		t.Lock()
		delete(t.queues, id)
		t.Unlock()
	}()

	log.Infof("node:%v process raft rpc queue", id)
	addr, err := t.nodesCache.GetNodeAddress(id)
	if err == cache.ErrNotFound {
		log.Errorf("node:%v process raft rpc queue, err:%v", id, err)
		return
	}

	client, conn, err := meta.NewRaftClient(addr.String())
	if err != nil {
		log.Errorf("new raft rpc client addr:%v err:%v", addr, err)
		return
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer conn.Close()
	defer cancel()

	streams := make([]meta.RaftMessage_RaftMessageClient, 2)
	for i := range streams {
		stream, err := client.RaftMessage(ctx)
		if err != nil {
			log.Errorf("failed to Raft transport stream to node %d at %s: %s", id, addr, err)
			return
		}
		streams[i] = stream
	}

	errCh := make(chan error, 1)
	t.stopper.RunTask(func() {
		for i := range streams {
			stream := streams[i]
			t.stopper.RunWorker(func() {
				errCh <- stream.RecvMsg(new(meta.RaftMessageResponse))
			})
		}
	})

	snapStream := streams[0]
	restStream := streams[1]

	timer := time.NewTimer(raftIdleTimeout)
	defer timer.Stop()
	for {
		timer.Reset(raftIdleTimeout)
		select {
		case <-t.stopper.ShouldStop():
			return
		case <-timer.C:
			log.Infof("closing Raft transport to %d at %s due to inactivity", id, addr)
			return
		case err := <-errCh:
			log.Infof("node %d addr %s closed raft transport, error:%v", id, addr, err)
			return
		case req := <-ch:
			breq := &meta.RaftMessageBatchRequest{Requests: []meta.RaftMessageRequest{*req}}
			if req.Message.Type == raftpb.MsgSnap {
				t.stopper.RunWorker(func() {
					err := snapStream.Send(breq)
					if err != nil {
						log.Errorf("failed to send raft snapshot to node:%v addr:%v err:%v", id, addr, err)
					} else {
						log.Infof("success to send raft snapshot to node:%v addr:%v", id, addr)
					}

					t.snapStatusChan <- &raftSnapStatus{req, err}
				})
			} else {
				for i := 0; i < raftMaxBatchRequestSize; i++ {
					select {
					case req := <-ch:
						breq.Requests = append(breq.Requests, *req)
					default:
						i = raftMaxBatchRequestSize
					}
				}

				if err := restStream.Send(breq); err != nil {
					log.Errorf("send to node: %v request error:%v", id, err)
					return
				}
			}
		}
	}
}

// send a message to the recipient specified in the request.
func (t *transport) send(req *meta.RaftMessageRequest) error {
	t.Lock()
	ch, ok := t.queues[req.ToNodeID]
	if !ok {
		ch = make(chan *meta.RaftMessageRequest, raftSendBufferSize)
		t.queues[req.ToNodeID] = ch
		go t.processQueue(req.ToNodeID)
	}
	t.Unlock()

	select {
	case ch <- req:
	default:
		return fmt.Errorf("queue for node %d is full", req.Message.To)
	}
	return nil
}
