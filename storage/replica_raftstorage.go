package storage

import (
	"bytes"
	"fmt"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/storage/engine"
	"github.com/taorenhai/ancestor/util"
)

// All calls to raft.RawNode. All of the functions exposed via the raft.Storage
// interface will in turn be called from RawNode.

// InitialState implements the raft.Storage interface.
func (r *replica) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
	var cs raftpb.ConfState

	hs, err := r.loadHardState(raftHardStateKey(r.rangeID))
	if err != nil {
		return raftpb.HardState{}, cs, err
	}

	if !raft.IsEmptyHardState(hs) {
		for _, rep := range r.rangeDesc.Replicas {
			cs.Nodes = append(cs.Nodes, uint64(rep.NodeID)+uint64(r.rangeID))
		}
	}

	return hs, cs, nil
}

// Entries implements the raft.Storage interface.
func (r *replica) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	if lo >= hi || maxSize < 0 {
		return nil, fmt.Errorf("range:%v, lo: %d, hi: %d, maxSize: %d, Entries's "+
			"parameter's invalid", r.rangeID, lo, hi, maxSize)
	}

	offseti := r.raftOffsetIndex
	if lo < offseti {
		return nil, raft.ErrCompacted
	}

	skey := raftLogKey(r.rangeID, lo)
	ekey := raftLogKey(r.rangeID, hi-1)
	var ents []raftpb.Entry
	expectedi := lo

	it := r.store.engine.NewIterator()
	defer it.Close()

	it.Seek(skey)
	for ; it.Valid(); it.Next() {
		var ent raftpb.Entry
		if bytes.Compare(it.Key(), meta.MVCCKey(ekey)) > 0 {
			break
		}
		if err := ent.Unmarshal(it.Value()); err != nil {
			return nil, err
		}

		ents = append(ents, ent)
		expectedi++
	}

	if len(ents) == int(hi)-int(lo) {
		return ents, nil
	}

	if len(ents) > 0 {
		if ents[0].Index > lo {
			return nil, raft.ErrCompacted
		}

		lasti, err := r.LastIndex()
		if err != nil {
			return nil, err
		}
		if lasti <= expectedi {
			return nil, raft.ErrUnavailable
		}

		return nil, fmt.Errorf("there is a gap in the index record between "+
			"lo:%d and hi:%d at index:%d", lo, hi, expectedi)
	}

	return ents, nil
}

// Term implements the raft.Storage interface.
func (r *replica) Term(i uint64) (uint64, error) {
	ents, err := r.Entries(i, i+1, 0)
	if err != nil {
		return 0, err
	}

	if len(ents) == 0 {
		if i == r.raftTruncateIndex {
			return r.raftTruncateTerm, nil
		}
		return 0, nil
	}

	return ents[0].Term, nil
}

// FirstIndex implements the raft.Storage interface.
func (r *replica) FirstIndex() (uint64, error) {
	return r.raftOffsetIndex + 1, nil
}

// LastIndex implements the raft.Storage interface.
func (r *replica) LastIndex() (uint64, error) {
	return r.raftLastIndex, nil
}

// getOffsetIndex retrieves the applied index from the supplied engine.
func (r *replica) loadOffsetIndex() (uint64, error) {
	v, err := r.store.engine.Get(raftOffsetIndexKey(r.rangeID))
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

	offseti, err := val.GetInt()
	if err != nil {
		return 0, err
	}

	return uint64(offseti), nil
}

// setOffsetIndex persists a new applied index.
func (r *replica) setOffsetIndex(batch engine.Engine, index uint64) error {
	var v meta.Value
	v.SetInt(int64(index))

	val, err := v.Marshal()
	if err != nil {
		return err
	}
	r.raftOffsetIndex = index

	return batch.Put(raftOffsetIndexKey(r.rangeID), val)
}

// loadAppliedIndex retrieves the applied index from the supplied engine.
func (r *replica) loadAppliedIndex() (uint64, error) {
	v, err := r.store.engine.Get(raftAppliedIndexKey(r.rangeID))
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

	appliedi, err := val.GetInt()
	if err != nil {
		return 0, err
	}

	return uint64(appliedi), nil
}

// setAppliedIndex persists a new applied index.
func (r *replica) setAppliedIndex(batch engine.Engine, index uint64) error {
	var v meta.Value
	v.SetInt(int64(index))

	val, err := v.Marshal()
	if err != nil {
		return err
	}
	r.raftAppliedIndex = index

	return batch.Put(raftAppliedIndexKey(r.rangeID), val)
}

// loadLastIndex retrieves the last index from storage.
func (r *replica) loadLastIndex() (uint64, error) {
	lasti := uint64(0)
	val, err := r.store.engine.Get(raftLastIndexKey(r.rangeID))
	if err != nil {
		return 0, err
	}

	if val != nil {
		var v meta.Value
		if err := v.Unmarshal(val); err != nil {
			return 0, err
		}

		int64Lasti, err := v.GetInt()
		if err != nil {
			return 0, err
		}
		lasti = uint64(int64Lasti)
	}

	return lasti, nil
}

// setLastIndex persists a new last index.
func (r *replica) setLastIndex(batch engine.Engine, index uint64) error {
	var v meta.Value
	v.SetInt(int64(index))

	val, err := v.Marshal()
	if err != nil {
		return err
	}
	r.raftLastIndex = index

	return batch.Put(raftLastIndexKey(r.rangeID), val)
}

// Snapshot implements the raft.Storage interface.
func (r *replica) Snapshot() (raftpb.Snapshot, error) {
	select {
	case snap := <-r.raftSnapshotChan:
		return snap, nil
	default:
	}

	if !r.store.acquireRaftSnapshot() {
		return raftpb.Snapshot{}, raft.ErrSnapshotTemporarilyUnavailable
	}

	r.store.stopper.RunWorker(func() {
		snap, err := r.snapshot()
		if err != nil {
			log.Errorf("range:%v get snapshot, error", err)
			return
		}
		r.raftSnapshotChan <- snap
	})

	return raftpb.Snapshot{}, raft.ErrSnapshotTemporarilyUnavailable
}

func (r *replica) data() ([]byte, error) {
	it := r.store.engine.NewSnapshot().NewIterator()
	defer it.Close()

	var kvs meta.SnapshotKeyValues

	for it.Seek(r.rangeDesc.StartKey); it.Valid(); it.Next() {
		if bytes.Compare(it.Key(), r.rangeDesc.EndKey) >= 0 {
			break
		}

		kvs.KeyValues = append(kvs.KeyValues, meta.MVCCKeyValue{Key: it.Key(), Value: it.Value()})
	}

	return kvs.Marshal()
}

func (r *replica) snapshot() (raftpb.Snapshot, error) {
	var m raftpb.SnapshotMetadata
	var d []byte
	var err error

	defer r.store.releaseRaftSnapshot()

	// Read the range metadata from the snapshot instead of the members
	// of the Range struct because they might be changed concurrently.
	if m.Index, err = r.loadAppliedIndex(); err != nil {
		return raftpb.Snapshot{}, errors.Trace(err)
	}

	if m.Term, err = r.Term(m.Index); err != nil {
		return raftpb.Snapshot{}, errors.Annotatef(err, "index:%d", m.Index)
	}

	for node := range r.raftGroup.Status().Progress {
		m.ConfState.Nodes = append(m.ConfState.Nodes, node)
	}

	if d, err = r.data(); err != nil {
		return raftpb.Snapshot{}, errors.Trace(err)
	}

	return raftpb.Snapshot{Data: d, Metadata: m}, nil
}

// append the given entries to the raft log. Takes the previous value of r.lastIndex and
// returns a new value.
func (r *replica) append(ents []raftpb.Entry) error {
	if len(ents) == 0 {
		return nil
	}

	for _, ent := range ents {
		val, err := ent.Marshal()
		if err != nil {
			return errors.Trace(err)
		}
		if err = r.store.engine.Put(raftLogKey(r.rangeID, ent.Index), val); err != nil {
			return errors.Trace(err)
		}
	}

	// Delete the appended log entries which never committed.
	last := ents[len(ents)-1].Index
	for i := last + 1; i <= r.raftLastIndex; i++ {
		if err := r.store.engine.Clear(raftLogKey(r.rangeID, i)); err != nil {
			return errors.Trace(err)
		}
	}

	return r.setLastIndex(r.store.engine, last)
}

func (r *replica) clear() error {
	it := r.store.engine.NewIterator()
	defer it.Close()

	for it.Seek(r.rangeDesc.StartKey); it.Valid(); it.Next() {
		if bytes.Compare(it.Key(), r.rangeDesc.EndKey) >= 0 {
			break
		}

		if err := r.store.engine.Clear(it.Key()); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// applySnapshot updates the replica based on the given snapshot.
// Returns the new last index.
func (r *replica) applySnapshot(s raftpb.Snapshot) error {
	kvs := &meta.SnapshotKeyValues{}

	if err := kvs.Unmarshal(s.Data); err != nil {
		return errors.Trace(err)
	}

	// Delete everything in the range and recreate it from the snapshot.
	if err := r.clear(); err != nil {
		return errors.Trace(err)
	}

	r.store.rangeStats[r.rangeID].RangeStatsInfo.TotalBytes = 0
	r.store.rangeStats[r.rangeID].RangeStatsInfo.TotalCount = 0

	// Write the snapshot into the range.
	for _, kv := range kvs.KeyValues {
		r.store.rangeStats[r.rangeID].UpdateRangeStats(0, 0, len(kv.Key), len(kv.Value))
		if err := r.store.engine.Put(kv.Key, kv.Value); err != nil {
			return errors.Trace(err)
		}
	}

	if err := r.setLastIndex(r.store.engine, s.Metadata.Index); err != nil {
		return errors.Trace(err)
	}

	if err := r.setAppliedIndex(r.store.engine, s.Metadata.Index); err != nil {
		return errors.Trace(err)
	}

	r.raftTruncateIndex = s.Metadata.Index
	r.raftTruncateTerm = s.Metadata.Term

	return nil
}

// setHardState persists the raft HardState. It is mainly used to restart
// the node with previous hard state
func (r *replica) setHardState(hs *raftpb.HardState) error {
	val, err := hs.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	return r.store.engine.Put(raftHardStateKey(r.rangeID), val)
}

// getHardState gets the raft HardState from rocksdb storage,  which mainly used
// to restart node from previous state
func (r *replica) loadHardState(key meta.MVCCKey) (raftpb.HardState, error) {
	var hs raftpb.HardState

	val, err := r.store.engine.Get(key)
	if err != nil {
		return hs, err
	}

	if val != nil {
		if err := hs.Unmarshal(val); err != nil {
			return hs, err
		}
	}

	return hs, nil
}

func (r *replica) compact(index uint64) error {
	offseti := r.raftOffsetIndex
	lasti := r.raftLastIndex

	if index <= offseti {
		return fmt.Errorf("range:%v offset:%v no need compact index:%v", r.rangeID, offseti, index)
	}
	if index > lasti {
		return fmt.Errorf("range:%v out of lastIndex: %v", r.rangeID, lasti)
	}

	skey := raftLogKey(r.rangeID, 0)
	ekey := raftLogKey(r.rangeID, index)
	batch := r.store.engine.NewBatch()
	if err := batch.Iterate(skey, ekey, func(kv meta.MVCCKeyValue) (bool, error) {
		return false, batch.Clear(kv.Key)
	}); err != nil {
		return err
	}

	if err := r.setOffsetIndex(batch, index); err != nil {
		return err
	}

	if err := batch.Commit(); err != nil {
		return err
	}

	return nil
}

// encoding state information, stores it in last part of rocksdb
func raftHardStateKey(id meta.RangeID) meta.MVCCKey {
	return meta.NewMVCCKey(meta.NewKey(util.HardStatePrefix, []byte(fmt.Sprintf("%v", id))))
}

func raftLogKey(id meta.RangeID, index uint64) meta.MVCCKey {
	return meta.NewMVCCKey(meta.NewKey(util.LogKeyPrefix, []byte(fmt.Sprintf("%.16d_%.16d", id, index))))
}

func raftAppliedIndexKey(id meta.RangeID) meta.MVCCKey {
	return meta.NewMVCCKey(meta.NewKey(util.AppliedIndexPrefix, []byte(fmt.Sprintf("%v", id))))
}

func raftLastIndexKey(id meta.RangeID) meta.MVCCKey {
	return meta.NewMVCCKey(meta.NewKey(util.LastIndexPrefix, []byte(fmt.Sprintf("%v", id))))
}

func raftOffsetIndexKey(id meta.RangeID) meta.MVCCKey {
	return meta.NewMVCCKey(meta.NewKey(util.OffsetIndexPrefix, []byte(fmt.Sprintf("%v", id))))
}
