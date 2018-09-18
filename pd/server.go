package pd

import (
	"encoding/json"
	"net"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/zssky/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
	"github.com/taorenhai/ancestor/util/cache"
	"github.com/taorenhai/ancestor/util/stop"
)

const (
	etcdTimeout = time.Second * 10
)

// Server is the pd server.
type Server struct {
	cfg *Config

	etcdClient *clientv3.Client

	isReady int64

	isMaster int64

	masterPath string
	// Master value saved in etcd masterhost key.
	// Every write will use this to check master validation.
	masterValue string

	newIDPath     string
	timestampPath string

	ts            atomic.Value
	lastSavedTime time.Time

	dm      *decisionMaker
	stopper *stop.Stopper

	delayRecord map[meta.RangeID]*delayRecord

	sync.RWMutex // Protects the following fields:

	cluster *cluster
	region  *region

	idAllocator *idAllocator

	nodeClientCache map[string]*meta.NodeClient

	debug bool
}

type masterHost struct {
	Master string
}

// NewServer creates the pd server with given configuration.
func NewServer(cfg *Config, debug bool) (*Server, error) {
	cfg.check()

	log.Infof("create etcd v3 client with endpoints %v", cfg.EtcdHosts)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.EtcdHosts,
		DialTimeout: etcdTimeout,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	mv, err := json.Marshal(masterHost{Master: cfg.Host})
	if err != nil {
		// can't fail, so panic here.
		log.Fatalf("marshal master err %v", err)
	}

	s := &Server{
		cfg:             cfg,
		etcdClient:      cli,
		masterPath:      path.Join(util.ETCDRootPath, "MasterHost"),
		masterValue:     string(mv),
		newIDPath:       path.Join(util.ETCDRootPath, "NewID"),
		timestampPath:   path.Join(util.ETCDRootPath, "Timestamp"),
		delayRecord:     make(map[meta.RangeID]*delayRecord),
		nodeClientCache: make(map[string]*meta.NodeClient),
		dm:              newDecisionMaker(),
		stopper:         stop.NewStopper(),
		debug:           debug,
	}

	lis, err := net.Listen("tcp", cfg.Host)
	if err != nil {
		log.Fatalf("Listen host error:%s", err.Error())
		return nil, errors.Trace(err)
	}
	grpcSrv := grpc.NewServer()
	meta.RegisterNodeMessageServer(grpcSrv, s)
	go grpcSrv.Serve(lis)

	s.cluster = newCluster(s)
	s.idAllocator = newIDAllocator(s)
	s.region = newRegion(s)

	if debug {
		startWebAPI(s)
	}

	return s, nil
}

// Start runs the pd server
func (s *Server) Start() error {
	log.Infof("pd server start")

	s.masterLoop()
	s.checkLoop()

	<-s.stopper.ShouldStop()
	log.Info("pd server stop")

	return nil
}

// Stop stops the pd server.
func (s *Server) Stop() {
	log.Info("stoping server")

	s.setIsMaster(false)
	s.stopper.Stop()
}

func (s *Server) handleGetTimestamp() (meta.Response, error) {
	resp := &meta.GetTimestampResponse{}
	ts, err := s.getTimestamp()
	if err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}

	resp.Timestamp = ts

	return resp, nil
}

func (s *Server) handleGetNewID() (meta.Response, error) {
	resp := &meta.GetNewIDResponse{}
	id, err := s.idAllocator.newID()
	if err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}

	resp.ID = meta.RangeID(id)

	return resp, nil
}

func (s *Server) handleGetRangeDescriptors(req *meta.GetRangeDescriptorsRequest) (meta.Response, error) {
	resp := &meta.GetRangeDescriptorsResponse{}
	rds, err := s.region.getRangeDescriptors(req.Key)
	if err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}

	resp.RangeDescriptor = rds

	return resp, nil
}

func (s *Server) handleSetRangeDescriptors(req *meta.SetRangeDescriptorsRequest) (meta.Response, error) {
	resp := &meta.SetRangeDescriptorsResponse{}
	if err := s.region.setRangeDescriptors(req.RangeDescriptor...); err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}
	log.Debugf("new ranges:%+v", req.RangeDescriptor)

	return resp, nil
}

func (s *Server) handleGetNodeDescriptors(req *meta.GetNodeDescriptorsRequest) (meta.Response, error) {
	resp := &meta.GetNodeDescriptorsResponse{}
	nds := s.cluster.getNodes(req.NodeIDs...)
	resp.NodeDescriptor = nds
	return resp, nil
}

func (s *Server) handleSetNodeDescriptors(req *meta.SetNodeDescriptorsRequest) (meta.Response, error) {
	resp := &meta.SetNodeDescriptorsResponse{}
	if err := s.cluster.setNodes(req.NodeDescriptor...); err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}
	log.Debugf("new nodes:%+v", req.NodeDescriptor)

	return resp, nil
}

func (s *Server) handleGetRangeStats(req *meta.GetRangeStatsRequest) (meta.Response, error) {
	return &meta.GetRangeStatsResponse{Rows: s.cluster.getRangeStats(req.RangeIDs...)}, nil
}

func (s *Server) handleSetRangeStats(req *meta.SetRangeStatsRequest) (meta.Response, error) {
	s.cluster.setRangeStats(req.RangeStatsInfo...)
	return &meta.SetRangeStatsResponse{}, nil
}

func (s *Server) handleGetNodeStats(req *meta.GetNodeStatsRequest) (meta.Response, error) {
	return &meta.GetNodeStatsResponse{Rows: s.cluster.getNodeStats(req.NodeIDs...)}, nil
}

func (s *Server) handleSetNodeStats(req *meta.SetNodeStatsRequest) (meta.Response, error) {
	s.cluster.setNodeStats(req.NodeStatsInfo)
	return &meta.SetNodeStatsResponse{}, nil
}

func (s *Server) handleLoadPdConf(req *meta.LoadPdConfRequest) (meta.Response, error) {
	resp := &meta.LoadPdConfResponse{}
	if err := s.LoadConfig(req.FilePath); err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}

	return resp, nil
}

func (s *Server) handleCreateReplica(req *meta.CreateReplicaRequest) (meta.Response, error) {
	resp := &meta.CreateReplicaResponse{}
	/*
		if err := s.addReplica(req.RangeID, req.NodeID); err != nil {
			resp.Header().Error = meta.NewError(err)
			return resp, errors.Trace(err)
		}
	*/

	return resp, nil
}

func (s *Server) handleRemoveReplica(req *meta.RemoveReplicaRequest) (meta.Response, error) {
	resp := &meta.RemoveReplicaResponse{}
	if err := s.deleteReplica(req.RangeID, req.NodeID); err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}

	return resp, nil
}

func (s *Server) handleTransferLeader(req *meta.TransferLeaderRequest) (meta.Response, error) {
	resp := &meta.TransferLeaderResponse{}
	if err := s.transferLeader(req.RangeID, req.From, req.To); err != nil {
		resp.Header().Error = meta.NewError(err)
		return resp, errors.Trace(err)
	}

	return resp, nil
}

func (s *Server) handleBootstrap(req *meta.BootstrapRequest) (meta.Response, error) {
	var err error
	resp := &meta.BootstrapResponse{}

	resp.NodeDescriptor, err = s.cluster.getNodeByAddr(req.Address)
	if err == nil {
		return resp, nil
	}

	if errors.Cause(err) == cache.ErrNotFound {
		if resp.NodeDescriptor, err = s.cluster.newNodeDescriptor(req.Address, req.Capacity); err == nil {
			if err = s.region.addSystemRangeDescriptor(resp.NodeDescriptor.NodeID); err == nil {
				return resp, nil
			}
		}
	}

	resp.Header().Error = meta.NewError(err)
	return resp, errors.Trace(err)
}

// executeCmd exec cmd
func (s *Server) executeCmd(inner interface{}) (meta.Response, error) {
	if !s.checkIsMaster() {
		return nil, errors.New("server is not master, please try again")
	}

	if !s.checkIsReady() {
		return nil, errors.New("server is not ready, please try again")
	}

	switch req := inner.(type) {
	case *meta.GetTimestampRequest:
		return s.handleGetTimestamp()
	case *meta.GetRangeDescriptorsRequest:
		return s.handleGetRangeDescriptors(req)
	case *meta.SetRangeDescriptorsRequest:
		return s.handleSetRangeDescriptors(req)
	case *meta.GetNodeDescriptorsRequest:
		return s.handleGetNodeDescriptors(req)
	case *meta.SetNodeDescriptorsRequest:
		return s.handleSetNodeDescriptors(req)
	case *meta.GetRangeStatsRequest:
		return s.handleGetRangeStats(req)
	case *meta.SetRangeStatsRequest:
		return s.handleSetRangeStats(req)
	case *meta.GetNodeStatsRequest:
		return s.handleGetNodeStats(req)
	case *meta.SetNodeStatsRequest:
		return s.handleSetNodeStats(req)
	case *meta.GetNewIDRequest:
		return s.handleGetNewID()
	case *meta.LoadPdConfRequest:
		return s.handleLoadPdConf(req)
	case *meta.CreateReplicaRequest:
		return s.handleCreateReplica(req)
	case *meta.RemoveReplicaRequest:
		return s.handleRemoveReplica(req)
	case *meta.TransferLeaderRequest:
		return s.handleTransferLeader(req)
	case *meta.BootstrapRequest:
		return s.handleBootstrap(req)
	default:
		log.Errorf("no found request type: %v", req)
		return nil, errors.New("no found request type")
	}
}

// HeartbeatMessage receive the heartbeat and send response.
func (s *Server) HeartbeatMessage(_ context.Context, _ *meta.HeartbeatRequest) (*meta.HeartbeatResponse, error) {
	return &meta.HeartbeatResponse{}, nil
}

// NodeMessage receive the message and execute the request
func (s *Server) NodeMessage(ctx context.Context, ba *meta.BatchRequest) (*meta.BatchResponse, error) {
	br := &meta.BatchResponse{}
	for _, req := range ba.GetReq() {
		inner := req.GetInner()
		resp, err := s.executeCmd(inner)
		if err != nil {
			log.Error(errors.ErrorStack(err))
			br.Header().Error = meta.NewError(errors.Trace(err))
		}
		if resp != nil {
			br.Add(resp)
		}
	}

	return br, nil
}

type master interface {
	put(clientv3.Cmp, ...clientv3.Op) error
	masterPut(...clientv3.Op) error
	getValues(string, ...clientv3.OpOption) ([]*mvccpb.KeyValue, error)
	newID() (int64, error)
}

func (s *Server) put(cmp clientv3.Cmp, ops ...clientv3.Op) error {
	return putValues(s.etcdClient, cmp, ops...)
}

func (s *Server) masterPut(ops ...clientv3.Op) error {
	return putValues(s.etcdClient, s.newMasterCmp(), ops...)
}

func (s *Server) getValues(key string, opts ...clientv3.OpOption) ([]*mvccpb.KeyValue, error) {
	return getValues(s.etcdClient, key, opts...)
}

func (s *Server) newID() (int64, error) {
	return s.idAllocator.newID()
}
