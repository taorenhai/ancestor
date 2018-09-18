package main

import (
	"bytes"
	"flag"
	"fmt"
	"runtime"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util"
	"github.com/taorenhai/ancestor/util/stop"
)

const (
	cacheSize = 8 << 20 // 8 MB
)

func getRangeIndex(eng engine.Engine, key meta.MVCCKey) (uint64, error) {
	v, err := eng.Get(key)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, nil
	}

	var val meta.Value
	if err = val.Unmarshal(v); err != nil {
		return 0, err
	}

	i, err := val.GetInt()
	if err != nil {
		return 0, err
	}

	return uint64(i), nil
}

func getRangeHardState(eng engine.Engine, key meta.MVCCKey) (raftpb.HardState, error) {
	var hs raftpb.HardState
	val, err := eng.Get(key)
	if err != nil {
		return hs, err
	}

	if val != nil {
		if err = hs.Unmarshal(val); err != nil {
			return hs, err
		}
	}

	return hs, err
}

func dumpRangeInfo(id int64, loc string) {
	stopper := stop.NewStopper()
	eng := engine.NewRocksDB(loc, cacheSize, stopper)
	if err := eng.Open(); err != nil {
		log.Errorf("could not create new rocksdb db instance: loc:%v err:%v\n", loc, err)
		return
	}

	offset, err := getRangeIndex(eng, meta.MVCCKey(meta.NewKey(util.OffsetIndexPrefix, []byte(fmt.Sprintf("%v", id)))))
	if err != nil {
		log.Errorf("getRangeIndex raftOffsetIndexKey error:%s", err.Error())
	}

	last, err := getRangeIndex(eng, meta.MVCCKey(meta.NewKey(util.LastIndexPrefix, []byte(fmt.Sprintf("%v", id)))))
	if err != nil {
		log.Errorf("getRangeIndex raftLastIndexKey error:%s", err.Error())
	}

	applied, err := getRangeIndex(eng, meta.MVCCKey(meta.NewKey(util.AppliedIndexPrefix, []byte(fmt.Sprintf("%v", id)))))
	if err != nil {
		log.Errorf("getRangeIndex raftLastIndexKey error:%s", err.Error())
	}

	hs, err := getRangeHardState(eng, meta.MVCCKey(meta.NewKey(util.HardStatePrefix, []byte(fmt.Sprintf("%v", id)))))
	if err != nil {
		log.Errorf("getRangeHardState error:%s", err.Error())
	}

	log.Infof("rangeID:%d, applied:%d, offset:%d, last:%d, HardState:%+v", id, applied, offset, last, hs)
}

func dumpUserData(key meta.MVCCKey, v []byte) {
	k, t, isValue, e := key.Decode()
	if e != nil {
		log.Warningf("MVCCDecodeKey key:%q error:%s", key, e.Error())
		return
	}

	if len(v) == 0 {
		log.Debugf("data mvcckey:%v key:%v not found", key, k)
		return
	}
	// data
	if isValue {
		val := &engine.MVCCValue{}
		if err := val.Unmarshal(v); err != nil {
			log.Warningf("Unmarshal data value error:%s", err.Error())
			return
		}

		if val.Value == nil {
			log.Errorf("data mvcckey:%v key:%v timestamp:%v, value is nil", key, k, t)
			return
		}
		if len(val.Value.Bytes) == 0 {
			log.Debugf("data mvcckey:%v key:%v timestamp:%v, deleted", key, k, t)
		}
		log.Debugf("data mvcckey:%v key:%v timestamp:%v, value:%q", key, k, t, val.Value.Bytes)
		return
	}

	// metadata
	val := &engine.MVCCMetadata{}
	if err := val.Unmarshal(v); err != nil {
		log.Warningf("Unmarshal metadata value error:%s", err.Error())
		return
	}

	log.Debugf("metadata mvcckey:%v, key:%v txnID:%v timestamp:%v, delete:%v", key, k, val.TransactionID, val.Timestamp, val.Deleted)
}

func dumpTransaction(val []byte) {
	txn := &meta.Transaction{}
	if err := txn.Unmarshal(val); err != nil {
		log.Warningf("Unmarshal txn value error:%s", err.Error())
		return
	}
	log.Debugf("Transaction ID:%s name:%s status:%v Priority:%d latestheart:%v timestamp:%v", txn.ID, txn.Name, txn.Status, txn.Priority, txn.LastHeartbeat, txn.Timestamp)
}

func dumpHardState(key meta.MVCCKey, val []byte) {
	if len(val) == 0 {
		log.Warningf("HardState key:%q val is nil", key)
		return
	}

	var hs raftpb.HardState
	if err := hs.Unmarshal(val); err != nil {
		log.Warningf("HardState Unmarshal val error:%s, key:%q", err.Error(), key)
		return
	}

	log.Debugf("HardState key:%q state:%+v", key, hs)
}

func dumpIndexValue(prefix string, key meta.MVCCKey, val []byte) {
	if len(val) == 0 {
		return
	}

	var v meta.Value
	if err := v.Unmarshal(val); err != nil {
		log.Warningf("key:%q, Unmarshal val error:%s", key, err.Error())
		return
	}

	i, err := v.GetInt()
	if err != nil {
		log.Warningf("key:%q GetInt error:%s", key, err.Error())
		return
	}

	log.Debugf("%s key:%q, value:%d", prefix, key, i)
}

// dumpRocksDB dump the data from RocksDB begin \x00 and end the nil or the count
func dumpRocksDB(searchKey meta.Key, loc string, count int, keyType string) {
	stopper := stop.NewStopper()
	eng := engine.NewRocksDB(loc, cacheSize, stopper)
	if err := eng.Open(); err != nil {
		log.Errorf("could not create new rocksdb db instance: loc:%v err:%v\n", loc, err)
		return
	}
	it := eng.NewIterator()
	defer it.Close()

	switch keyType {
	case "txn":
		it.Seek(meta.NewKey(util.TxnPrefix, []byte(searchKey)))
	case "user":
		it.Seek(meta.NewKey(util.UserPrefix, []byte(searchKey)))
	case "system":
		it.Seek(meta.NewKey(util.SystemPrefix, []byte(searchKey)))
	default:
		it.Seek([]byte(searchKey))
	}
	log.Infof("-------------------------------dump begin--------------------------")
	for ; it.Valid() && count > 0; it.Next() {
		key := it.Key()
		switch {
		case bytes.HasPrefix(key, util.TxnPrefix):
			dumpTransaction(it.Value())

		case bytes.HasPrefix(key, util.UserPrefix):
			dumpUserData(key, it.Value())

		case bytes.HasPrefix(key, util.HardStatePrefix):
			dumpHardState(key, it.Value())

		case bytes.HasPrefix(key, util.AppliedIndexPrefix):
			dumpIndexValue("AppliedIndex", key, it.Value())

		case bytes.HasPrefix(key, util.LastIndexPrefix):
			dumpIndexValue("LastIndex", key, it.Value())

		case bytes.HasPrefix(key, util.OffsetIndexPrefix):
			dumpIndexValue("OffsetIndex", key, it.Value())

		default:
			log.Debugf("data key:%q, value:%.30q", key, it.Value())
		}
		count--
	}
	log.Infof("-------------------------------dump end--------------------------")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var key = flag.String("k", "", "search key")
	var r = flag.Int64("r", -1, "get range info")
	var loc = flag.String("l", "", "local path")
	var count = flag.Int("c", 10, "seek count")
	var keyType = flag.String("t", "all", "key type: txn, user, system, all")

	flag.Parse()

	if *r == -1 {
		log.Debugf("search key:%s, type:%v, local path:%s, seek count:%d", *key, *keyType, *loc, *count)
		dumpRocksDB(meta.Key(*key), *loc, *count, *keyType)
		return
	}

	log.Debugf("get range info ID:%d path:%s", *r, *loc)
	dumpRangeInfo(*r, *loc)

}
