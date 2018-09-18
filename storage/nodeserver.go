package storage

import (
	"net"

	"github.com/juju/errors"
	"github.com/shirou/gopsutil/disk"
	"github.com/zssky/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/taorenhai/ancestor/client"
	"github.com/taorenhai/ancestor/config"
	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util/cache"
	"github.com/taorenhai/ancestor/util/stop"
)

const (
	cacheSize = 128 << 20 // 128 MB
)

// NodeServer - node server
type NodeServer struct {
	store   Store
	stopper *stop.Stopper
	config  nodeConfig
	client  client.Admin
	meta.NodeDescriptor
}

type nodeConfig struct {
	nodeHost  string
	etcdHosts string
	dataDir   string
}

// NewNodeServer create a NodeServer from cfg
func NewNodeServer(cfg *config.Config) (*NodeServer, error) {
	stopper := stop.NewStopper()
	if stopper == nil {
		return nil, errors.Errorf("create New Stopper failure")
	}

	nodeCfg := parseConfig(cfg)

	// create a new Client
	c, err := client.Open(nodeCfg.etcdHosts)
	if err != nil {
		return nil, errors.Trace(err)
	}
	admin := c.GetAdmin()

	// create a DBEngine
	eng, err := newDBEngine(nodeCfg.dataDir, stopper)
	if err != nil {
		return nil, errors.Trace(err)
	}

	nodes, err := admin.GetNodes()
	if err != nil {
		return nil, errors.Trace(err)
	}

	fs, err := disk.Usage(nodeCfg.dataDir)
	if err != nil {
		return nil, errors.Trace(err)
	}

	nd, err := admin.Bootstrap(nodeCfg.nodeHost, int64(fs.Free))
	if err != nil {
		return nil, errors.Trace(err)
	}

	// create NodesCache
	nc := cache.NewNodesCache()
	nc.AddNodes(nodes)

	serv := grpc.NewServer()
	// create a new Store
	store, err := NewStore(nd.NodeID, serv, admin, eng, stopper, nc)
	if err != nil {
		return nil, errors.Trace(err)
	}

	s := &NodeServer{
		config:         nodeCfg,
		stopper:        stopper,
		store:          store,
		client:         admin,
		NodeDescriptor: nd,
	}

	admin.SetLocalServer(s.NodeID, s)

	meta.RegisterNodeMessageServer(serv, s)
	lis, err := net.Listen("tcp", nodeCfg.nodeHost)
	if err != nil {
		log.Errorf("host:%v failed to listen: %v", nodeCfg.nodeHost, err)
		return nil, err
	}

	stopper.RunTask(func() {
		stopper.RunWorker(func() {
			if err := serv.Serve(lis); err != nil {
				log.Errorf("host:%v failed to server: %v", nodeCfg.nodeHost, err)
				return
			}
		})
	})

	return s, nil
}

// ExecuteCmd the batch ExecuteCmd interface
func (ns *NodeServer) ExecuteCmd(req *meta.BatchRequest) (*meta.BatchResponse, error) {
	resp, err := ns.store.Send(req)
	if err != nil {
		resp.Header().Error = meta.NewError(errors.Cause(err))
	}
	return resp, nil
}

// NodeMessage the grpc server CallBack function
func (ns *NodeServer) NodeMessage(_ context.Context, req *meta.BatchRequest) (*meta.BatchResponse, error) {
	return ns.ExecuteCmd(req)
}

// HeartbeatMessage the grpc server heartbeat function.
func (ns *NodeServer) HeartbeatMessage(_ context.Context, _ *meta.HeartbeatRequest) (*meta.HeartbeatResponse, error) {
	return &meta.HeartbeatResponse{}, nil
}

// listen implements the store.transport interface by registering a serverInterface

func parseConfig(cfg *config.Config) nodeConfig {
	var nc nodeConfig

	nc.nodeHost = cfg.GetString(config.SectionDefault, "NodeHost")
	nc.dataDir = cfg.GetString(config.SectionDefault, "DataDir")
	nc.etcdHosts = cfg.GetString(config.SectionDefault, "EtcdHosts")

	return nc
}

// Start start node server
func (ns *NodeServer) Start() error {
	log.Infof("node server start")
	defer ns.Close()

	rds, err := ns.client.GetRangeDescriptors()
	if err != nil {
		return errors.Trace(err)
	}

	if err = ns.store.Create(rds); err != nil {
		return errors.Trace(err)
	}

	<-ns.stopper.ShouldStop()
	return nil
}

// Close stop NodeServer
func (ns *NodeServer) Close() {
	ns.stopper.Stop()
}

// newDBEngine new a DB Engine
func newDBEngine(dataDir string, stopper *stop.Stopper) (engine.Engine, error) {
	var eng engine.Engine
	if len(dataDir) == 0 {
		eng = engine.NewInMem(cacheSize, stopper).RocksDB
	} else {
		eng = engine.NewRocksDB(dataDir, cacheSize, stopper)
	}

	if eng == nil {
		return nil, errors.Errorf("new Engine failure")
	}

	if err := eng.Open(); err != nil {
		return nil, errors.Errorf("could not create new db instance: dir:%v err:%v\n", dataDir, err)
	}
	return eng, nil
}
