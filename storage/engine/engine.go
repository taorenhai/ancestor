package engine

import (
	"sync"

	"github.com/gogo/protobuf/proto"

	"github.com/taorenhai/ancestor/meta"
)

// Iterator is an interface for iterating over key/value pairs in an
// engine. Iterator implementations are thread safe unless otherwise
// noted.
type Iterator interface {
	// Close frees up resources held by the iterator.
	Close()
	// Seek advances the iterator to the first key in the engine which
	// is >= the provided key.
	Seek(key []byte)
	// Valid returns true if the iterator is currently valid. An
	// iterator which hasn't been seeked or has gone past the end of the
	// key range is invalid.
	Valid() bool
	// Next advances the iterator to the next key/value in the
	// iteration. After this call, Valid() will be true if the
	// iterator was not positioned at the last key.
	Next()
	// Prev moves the iterator backward to the previous key/value
	// in the iteration. After this call, Valid() will be true if the
	// iterator was not positioned at the first key.
	Key() meta.MVCCKey
	// Value returns the current value as a byte slice.
	Value() []byte
	// ValueProto unmarshals the value the iterator is currently
	// pointing to using a protobuf decoder.
	ValueProto(msg proto.Message) error
	// unsafeKey returns the same value as Key, but the memory is invalidated on
	// the next call to {Next,Prev,Seek,SeekReverse,Close}.
	unsafeKey() meta.MVCCKey
	// unsafeKey returns the same value as Value, but the memory is invalidated
	// on the next call to {Next,Prev,Seek,SeekReverse,Close}.
	unsafeValue() []byte
	// Error returns the error, if any, which the iterator encountered.
	Error() error
}

// Engine is the interface that wraps the core operations of a
// key/value store.
type Engine interface {
	// Open initializes the engine.
	Open() error
	// Close closes the engine, freeing up any outstanding resources.
	Close()
	// Attrs returns the engine/store attributes.
	//Attrs() meta.Attributes
	// Put sets the given key to the value provided.
	Put(key meta.MVCCKey, value []byte) error
	// Get returns the value for the given key, nil otherwise.
	Get(key meta.MVCCKey) ([]byte, error)
	// GetProto fetches the value at the specified key and unmarshals it
	// using a protobuf decoder. Returns true on success or false if the
	// key was not found. On success, returns the length in bytes of the
	// key and the value.
	GetProto(key meta.MVCCKey, msg proto.Message) (ok bool, keyBytes, valBytes int64, err error)
	// Iterate scans from start to end keys, visiting at most max
	// key/value pairs. On each key value pair, the function f is
	// invoked. If f returns an error or if the scan itself encounters
	// an error, the iteration will stop and return the error.
	// If the first result of f is true, the iteration stops.
	Iterate(start, end meta.MVCCKey, f func(meta.MVCCKeyValue) (bool, error)) error
	// Clear removes the item from the db with the given key.
	// Note that clear actually removes entries from the storage
	// engine, rather than inserting tombstones.
	Clear(key meta.MVCCKey) error
	// Merge is a high-performance write operation used for values which are
	// accumulated over several writes. Multiple values can be merged
	// sequentially into a single key; a subsequent read will return a "merged"
	// value which is computed from the original merged values.
	//
	// Merge currently provides specialized behavior for three data types:
	// integers, byte slices, and time series observations. Merged integers are
	// summed, acting as a high-performance accumulator.  Byte slices are simply
	// concatenated in the order they are merged. Time series observations
	// (stored as byte slices with a special tag on the meta.Value) are
	// combined with specialized logic beyond that of simple byte slices.
	//
	// The logic for merges is written in db.cc in order to be compatible with RocksDB.
	//Merge(key meta.MVCCKey, value []byte) error
	// Capacity returns capacity details for the engine's available storage.
	//Capacity() (meta.StoreCapacity, error)
	// SetGCTimeouts sets timeout values for GC of transaction entries. The
	// values are specified in unix time in nanoseconds for the minimum
	// transaction row timestamp. Rows with timestamps less than the associated
	// value will be GC'd during compaction.
	SetGCTimeouts(minTxnTS int64)
	// Flush causes the engine to write all in-memory data to disk
	// immediately.
	Flush() error
	// NewIterator returns a new instance of an Iterator over this
	// engine. The caller must invoke Iterator.Close() when finished with
	// the iterator to free resources.
	NewIterator(reverse bool) Iterator
	// NewSnapshot returns a new instance of a read-only snapshot
	// engine. Snapshots are instantaneous and, as long as they're
	// released relatively quickly, inexpensive. Snapshots are released
	// by invoking Close(). Note that snapshots must not be used after the
	// original engine has been stopped.
	NewSnapshot() Engine
	// NewBatch returns a new instance of a batched engine which wraps
	// this engine. Batched engines accumulate all mutations and apply
	// them atomically on a call to Commit().
	NewBatch() Engine
	// Commit atomically applies any batched updates to the underlying
	// engine. This is a noop unless the engine was created via NewBatch().
	Commit() error
	// Defer adds a callback to be run after the batch commits
	// successfully.  If Commit() fails (or if this engine was not
	// created via NewBatch()), deferred callbacks are not called. As
	// with the defer statement, the last callback to be deferred is the
	// first to be executed.
	Defer(fn func())
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return proto.NewBuffer(nil)
	}, //func (r *rocksDBBatch) Capacity() (meta.StoreCapacity, error) {
	//	return r.parent.Capacity()
	//}

}

// PutProto sets the given key to the protobuf-serialized byte string
// of msg and the provided timestamp. Returns the length in bytes of
// key and the value.
func PutProto(engine Engine, key meta.MVCCKey, msg proto.Message) (keyBytes, valBytes int64, err error) {
	buf := bufferPool.Get().(*proto.Buffer)
	buf.Reset()

	if err = buf.Marshal(msg); err != nil {
		bufferPool.Put(buf)
		return
	}
	data := buf.Bytes()
	if err = engine.Put(key, data); err != nil {
		bufferPool.Put(buf)
		return
	}
	keyBytes = int64(len(key))
	valBytes = int64(len(data))

	bufferPool.Put(buf)
	return
}

// Scan returns up to max key/value objects starting from
// start (inclusive) and ending at end (non-inclusive).
// Specify max=0 for unbounded scans.
func Scan(engine Engine, start, end meta.MVCCKey, max int64) ([]meta.MVCCKeyValue, error) {
	var kvs []meta.MVCCKeyValue
	err := engine.Iterate(start, end, func(kv meta.MVCCKeyValue) (bool, error) {
		if max != 0 && int64(len(kvs)) >= max {
			return true, nil
		}
		kvs = append(kvs, kv)
		return false, nil
	})
	return kvs, err
}

// ClearRange removes a set of entries, from start (inclusive) to end
// (exclusive). This function returns the number of entries
// removed. Either all entries within the range will be deleted, or
// none, and an error will be returned. Note that this function
// actually removes entries from the storage engine, rather than
// inserting tombstones, as with deletion through the MVCC.
func ClearRange(engine Engine, start, end meta.MVCCKey) (int, error) {
	b := engine.NewBatch()
	defer b.Close()
	count := 0
	if err := engine.Iterate(start, end, func(kv meta.MVCCKeyValue) (bool, error) {
		if err := b.Clear(kv.Key); err != nil {
			return false, err
		}
		count++
		return false, nil
	}); err != nil {
		return 0, err
	}
	return count, b.Commit()
}
