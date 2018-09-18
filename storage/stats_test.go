package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util/stop"
	"github.com/zssky/log"
)

const (
	cacheSize = 128 << 20

	dataDir = "/tmp/nodedb9011"
)

var (
	testNodeID  = meta.NodeID(1)
	testRangeID = meta.RangeID(10)

	rStatsInfo = &meta.RangeStatsInfo{
		NodeID:     testNodeID,
		RangeID:    testRangeID,
		TotalBytes: 100,
		TotalCount: 20,
	}

	mapRangeStats = map[meta.RangeID]*meta.RangeStatsInfo{
		testRangeID: rStatsInfo,
	}

	nodeStats = meta.NodeStatsInfo{
		NodeID:         testNodeID,
		TotalBytes:     2900,
		TotalCount:     10,
		LeaderCount:    5,
		RangeStatsInfo: mapRangeStats,
	}

	desc = meta.RangeDescriptor{
		RangeID:  testRangeID,
		StartKey: meta.Key("t"),
		EndKey:   meta.Key("z"),
		Replicas: []meta.ReplicaDescriptor{
			{NodeID: testNodeID},
		},
	}
)

func TestStoreRangeStats(t *testing.T) {
	eng, err := testNewEngine(dataDir, int64(cacheSize))
	if err != nil {
		log.Error(err)
		return
	}

	store := testNewStore(testNodeID, eng)

	if err := store.storeNodeStats(&nodeStats); err != nil {
		log.Error(err)
		return
	}

	log.Infof("test store range stats success")
}

func TestLoadRangeStats(t *testing.T) {
	eng, err := testNewEngine(dataDir, int64(cacheSize))
	if err != nil {
		log.Error(err)
		return
	}

	store := testNewStore(testNodeID, eng)
	store.addRangeStats(testRangeID)

	stats, err := store.loadRangeStats()
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("test load range stats success, stats:%v", stats)
}

func TestNewNodeStats(t *testing.T) {
	eng, err := testNewEngine(dataDir, int64(cacheSize))
	if err != nil {
		log.Error(err)
		return
	}

	store := testNewStore(testNodeID, eng)
	store.createReplica(desc)
	time.Sleep(10 * time.Second)
	store.newNodeStats()

	log.Infof("test upload range stats success")
}

func testNewStore(nodeID meta.NodeID, eng engine.Engine) *store {
	return &store{
		nodeID:     nodeID,
		engine:     eng,
		replicas:   make(map[meta.RangeID]*replica),
		rangeStats: make(map[meta.RangeID]*meta.RangeStats),
		stopper:    stop.NewStopper(),
	}
}

func testNewEngine(dataDir string, cs int64) (engine.Engine, error) {
	stopper := stop.NewStopper()
	if stopper == nil {
		return nil, fmt.Errorf("create New Stopper failure")
	}

	eng := engine.NewRocksDB(dataDir, cs, stopper)
	if eng == nil {
		return nil, fmt.Errorf("new rocksdb error")
	}
	log.Infof("new engine success")
	if err := eng.Open(); err != nil {
		return nil, fmt.Errorf("new engine open error:%v", err)
	}
	log.Infof("new engine open success")

	return eng, nil
}
