package engine

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/juju/errors"
	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/stop"
	"github.com/zssky/log"
)

// Logger is a logging function to be set by the importing package. Its
// presence allows us to avoid depending on a logging package.
var Logger = func(string, ...interface{}) {}

// RocksDB is a wrapper around a RocksDB database instance.
type RocksDB struct {
	rdb         *badger.DB
	dir         string // The data directory
	cacheSize   int64  // Memory to use to cache values.
	stopper     *stop.Stopper
	deallocated chan struct{} // Closed when the underlying handle is deallocated.
}

// NewRocksDB allocates and returns a new RocksDB object.
func NewRocksDB(dir string, cacheSize int64, stopper *stop.Stopper) *RocksDB {
	if dir == "" {
		panic(fmt.Errorf("dir must be non-empty"))
	}
	return &RocksDB{
		dir:         dir,
		cacheSize:   cacheSize,
		stopper:     stopper,
		deallocated: make(chan struct{}),
	}
}

// Open creates options and opens the database. If the database
// doesn't yet exist at the specified directory, one is initialized
// from scratch. The RocksDB Open and Close methods are reference
// counted such that subsequent Open calls to an already opened
// RocksDB instance only bump the reference count. The RocksDB is only
// closed when a sufficient number of Close calls are performed to
// bring the reference count down to 0.
func (r *RocksDB) Open() error {
	if r.rdb != nil {
		return nil
	}

	if len(r.dir) != 0 {
		log.Infof("opening rocksdb instance at %q\n", r.dir)
	}

	opt := badger.DefaultOptions
	opt.Dir = fmt.Sprintf("%s/data/", r.dir)
	opt.ValueDir = fmt.Sprintf("%s/values/", r.dir)

	db, err := badger.Open(opt)
	if err != nil {
		return errors.Trace(err)
	}

	r.rdb = db

	// Start a gorountine that will finish when the underlying handle
	// is deallocated. This is used to check a leak in tests.
	go func() {
		<-r.deallocated
	}()
	r.stopper.AddCloser(r)
	return nil
}

// Close closes the database by deallocating the underlying handle.
func (r *RocksDB) Close() {
	if r.rdb == nil {
		log.Errorf("closing unopened rocksdb instance")
		return
	}
	if len(r.dir) == 0 {
		log.Infof("closing in-memory rocksdb instance")
	} else {
		log.Infof("closing rocksdb instance at %q\n", r.dir)
	}
	if r.rdb != nil {
		r.rdb.Close()
		r.rdb = nil
	}
	close(r.deallocated)
}

func emptyKeyError() error {
	return fmt.Errorf("attempted access to empty key")
}

// Put sets the given key to the value provided.
//
// The key and value byte slices may be reused safely. put takes a copy of
// them before returning.
func (r *RocksDB) Put(key meta.MVCCKey, value []byte) error {
	if len(key) == 0 {
		return emptyKeyError()
	}

	return r.rdb.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// Get returns the value for the given key.
func (r *RocksDB) Get(key meta.MVCCKey) ([]byte, error) {
	return r.getInternal(key, nil)
}

// getInternal returns the value for the given key.
func (r *RocksDB) getInternal(key meta.MVCCKey, txn *badger.Txn) ([]byte, error) {
	if len(key) == 0 {
		return nil, emptyKeyError()
	}

	if txn == nil {
		txn = r.rdb.NewTransaction(false)
	}

	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, errors.Annotatef(err, "key:%v", key)
	}

	return item.Value()
}

// GetProto fetches the value at the specified key and unmarshals it.
func (r *RocksDB) GetProto(key meta.MVCCKey, msg proto.Message) (ok bool, keyBytes, valBytes int64, err error) {
	return r.getProtoInternal(key, msg, nil)
}

func (r *RocksDB) getProtoInternal(key meta.MVCCKey, msg proto.Message, txn *badger.Txn) (ok bool, keyBytes, valBytes int64, err error) {
	if len(key) == 0 {
		err = emptyKeyError()
		return
	}

	data, e := r.getInternal(key, txn)
	if e != nil {
		err = errors.Trace(e)
		return
	}

	ok = true
	if msg != nil {
		err = proto.Unmarshal(data, msg)
	}
	keyBytes = int64(len(key))
	valBytes = int64(len(data))
	return
}

// Clear removes the item from the db with the given key.
func (r *RocksDB) Clear(key meta.MVCCKey) error {
	if len(key) == 0 {
		return emptyKeyError()
	}
	return r.rdb.NewTransaction(true).Delete(key)
}

// Iterate iterates from start to end keys, invoking f on each
// key/value pair. See engine.Iterate for details.
func (r *RocksDB) Iterate(start, end meta.MVCCKey, f func(meta.MVCCKeyValue) (bool, error)) error {
	return r.iterateInternal(start, end, f, nil)
}

func (r *RocksDB) iterateInternal(start, end meta.MVCCKey, f func(meta.MVCCKeyValue) (bool, error),
	snapshotHandle *badger.Txn) error {
	if bytes.Compare(start, end) >= 0 {
		return nil
	}
	it := newRocksDBIterator(r.rdb, snapshotHandle, false)
	defer it.Close()

	it.Seek(start)
	for ; it.Valid(); it.Next() {
		k := it.Key()
		if !it.Key().Less(end) {
			break
		}
		if done, err := f(meta.MVCCKeyValue{Key: k, Value: it.Value()}); done || err != nil {
			return err
		}
	}
	// Check for any errors during iteration.
	return it.Error()
}

// SetGCTimeouts calls through to the DBEngine's SetGCTimeouts method.
func (r *RocksDB) SetGCTimeouts(minTxnTS int64) {
	//TODO
}

// Destroy destroys the underlying filesystem data associated with the database.
func (r *RocksDB) Destroy() error {
	//TODO

	return nil
}

// Flush causes RocksDB to write all in-memory data to disk immediately.
func (r *RocksDB) Flush() error {
	//TODO
	return nil
}

// NewIterator returns an iterator over this rocksdb engine.
func (r *RocksDB) NewIterator(reverse bool) Iterator {
	return newRocksDBIterator(r.rdb, nil, reverse)
}

// NewSnapshot creates a snapshot handle from engine and returns a
// read-only rocksDBSnapshot engine.
func (r *RocksDB) NewSnapshot() Engine {
	if r.rdb == nil {
		panic("RocksDB is not initialized yet")
	}
	return &rocksDBSnapshot{
		parent: r,
		handle: r.rdb.NewTransaction(false),
	}
}

// NewBatch returns a new batch wrapping this rocksdb engine.
func (r *RocksDB) NewBatch() Engine {
	return newRocksDBBatch(r)
}

// Commit is a noop for RocksDB engine.
func (r *RocksDB) Commit() error {
	return nil
}

// Defer is not implemented for RocksDB engine.
func (r *RocksDB) Defer(func()) {
	panic("only implemented for rocksDBBatch")
}

type rocksDBSnapshot struct {
	parent *RocksDB
	handle *badger.Txn
}

// Open is a noop.
func (r *rocksDBSnapshot) Open() error {
	return nil
}

// Close releases the snapshot handle.
func (r *rocksDBSnapshot) Close() {
	r.handle.Discard()
}

// Put is illegal for snapshot and returns an error.
func (r *rocksDBSnapshot) Put(key meta.MVCCKey, value []byte) error {
	return fmt.Errorf("cannot Put to a snapshot")
}

// Get returns the value for the given key, nil otherwise using
// the snapshot handle.
func (r *rocksDBSnapshot) Get(key meta.MVCCKey) ([]byte, error) {
	return r.parent.getInternal(key, r.handle)
}

func (r *rocksDBSnapshot) GetProto(key meta.MVCCKey, msg proto.Message) (
	ok bool, keyBytes, valBytes int64, err error) {
	return r.parent.getProtoInternal(key, msg, r.handle)
}

// Iterate iterates over the keys between start inclusive and end
// exclusive, invoking f() on each key/value pair using the snapshot
// handle.
func (r *rocksDBSnapshot) Iterate(start, end meta.MVCCKey, f func(meta.MVCCKeyValue) (bool, error)) error {
	return r.parent.iterateInternal(start, end, f, r.handle)
}

// Clear is illegal for snapshot and returns an error.
func (r *rocksDBSnapshot) Clear(key meta.MVCCKey) error {
	return fmt.Errorf("cannot Clear from a snapshot")
}

// Merge is illegal for snapshot and returns an error.
/*func (r *rocksDBSnapshot) Merge(key meta.MVCCKey, value []byte) error {
	return fmt.Errorf("cannot Merge to a snapshot")
}*/

// SetGCTimeouts is a noop for a snapshot.
func (r *rocksDBSnapshot) SetGCTimeouts(minTxnTS int64) {
}

// Flush is a no-op for snapshots.
func (r *rocksDBSnapshot) Flush() error {
	return nil
}

// NewIterator returns a new instance of an Iterator over the
// engine using the snapshot handle.
func (r *rocksDBSnapshot) NewIterator(reverse bool) Iterator {
	return newRocksDBIterator(r.parent.rdb, r.handle, reverse)
}

// NewSnapshot is illegal for snapshot.
func (r *rocksDBSnapshot) NewSnapshot() Engine {
	panic("cannot create a NewSnapshot from a snapshot")
}

// NewBatch is illegal for snapshot.
func (r *rocksDBSnapshot) NewBatch() Engine {
	panic("cannot create a NewBatch from a snapshot")
}

// Commit is illegal for snapshot and returns an error.
func (r *rocksDBSnapshot) Commit() error {
	return fmt.Errorf("cannot Commit to a snapshot")
}

// Defer is not implemented for rocksDBSnapshot.
func (r *rocksDBSnapshot) Defer(func()) {
	panic("only implemented for rocksDBBatch")
}

type rocksDBBatch struct {
	parent *RocksDB
	batch  *badger.Txn
	defers []func()
}

func newRocksDBBatch(r *RocksDB) *rocksDBBatch {
	return &rocksDBBatch{
		parent: r,
		batch:  r.rdb.NewTransaction(true),
	}
}

func (r *rocksDBBatch) Open() error {
	return fmt.Errorf("cannot open a batch")
}

func (r *rocksDBBatch) Close() {
	if r.batch != nil {
		r.batch.Discard()
		r.batch = nil
	}
}

func (r *rocksDBBatch) Put(key meta.MVCCKey, value []byte) error {
	if len(key) == 0 {
		return emptyKeyError()
	}
	return r.batch.Set(key, value)
}

func (r *rocksDBBatch) Get(key meta.MVCCKey) ([]byte, error) {
	if len(key) == 0 {
		return nil, emptyKeyError()
	}
	item, err := r.batch.Get(key)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return item.Value()
}

func (r *rocksDBBatch) GetProto(key meta.MVCCKey, msg proto.Message) (ok bool, keyBytes, valBytes int64, err error) {
	if len(key) == 0 {
		err = emptyKeyError()
		return
	}
	data, err := r.Get(key)
	if err != nil {
		err = errors.Trace(err)
		return
	}

	ok = true
	if msg != nil {
		err = proto.Unmarshal(data, msg)
	}
	keyBytes = int64(len(key))
	valBytes = int64(len(data))
	return
}

func (r *rocksDBBatch) Iterate(start, end meta.MVCCKey, f func(meta.MVCCKeyValue) (bool, error)) error {
	if bytes.Compare(start, end) >= 0 {
		return nil
	}
	it := &rocksDBIterator{
		iter: r.batch.NewIterator(badger.IteratorOptions{}),
	}
	defer it.Close()

	it.Seek(start)
	for ; it.Valid(); it.Next() {
		k := it.Key()
		if !it.Key().Less(end) {
			break
		}
		if done, err := f(meta.MVCCKeyValue{Key: k, Value: it.Value()}); done || err != nil {
			return err
		}
	}
	// Check for any errors during iteration.
	return it.Error()
}

func (r *rocksDBBatch) Clear(key meta.MVCCKey) error {
	if len(key) == 0 {
		return emptyKeyError()
	}
	return r.batch.Delete(key)
}

func (r *rocksDBBatch) SetGCTimeouts(minTxnTS int64) {
	// no-op
}

func (r *rocksDBBatch) Flush() error {
	return fmt.Errorf("cannot flush a batch")
}

func (r *rocksDBBatch) NewIterator(reverse bool) Iterator {
	return &rocksDBIterator{
		iter: r.batch.NewIterator(badger.IteratorOptions{
			Reverse: reverse,
		}),
	}
}

func (r *rocksDBBatch) NewSnapshot() Engine {
	panic("cannot create a NewSnapshot from a batch")
}

func (r *rocksDBBatch) NewBatch() Engine {
	return newRocksDBBatch(r.parent)
}

func (r *rocksDBBatch) Commit() error {
	if r.batch == nil {
		panic("this batch was already committed")
	}
	if err := r.batch.Commit(nil); err != nil {
		return errors.Trace(err)
	}

	r.batch = nil

	// On success, run the deferred functions in reverse order.
	for i := len(r.defers) - 1; i >= 0; i-- {
		r.defers[i]()
	}
	r.defers = nil

	return nil
}

func (r *rocksDBBatch) Defer(fn func()) {
	r.defers = append(r.defers, fn)
}

type rocksDBIterator struct {
	iter  *badger.Iterator
	valid bool
}

// newRocksDBIterator returns a new iterator over the supplied RocksDB
// instance. If txn is not nil, uses the indicated snapshot.
// The caller must call rocksDBIterator.Close() when finished with the
// iterator to free up resources.
func newRocksDBIterator(rdb *badger.DB, txn *badger.Txn, reverse bool) *rocksDBIterator {
	if txn != nil {
		return &rocksDBIterator{
			iter: txn.NewIterator(badger.IteratorOptions{
				Reverse: reverse,
			}),
		}
	}
	// In order to prevent content displacement, caching is disabled
	// when performing scans. Any options set within the shared read
	// options field that should be carried over needs to be set here
	// as well.
	return &rocksDBIterator{
		iter: rdb.NewTransaction(true).NewIterator(badger.IteratorOptions{}),
	}
}

// The following methods implement the Iterator interface.
func (r *rocksDBIterator) Close() {
	r.iter.Close()
}

func (r *rocksDBIterator) Seek(key []byte) {
	r.iter.Seek(key)
}

func (r *rocksDBIterator) Valid() bool {
	return r.iter.Valid()
}

func (r *rocksDBIterator) Next() {
	r.iter.Next()
}

func (r *rocksDBIterator) Key() meta.MVCCKey {
	return meta.MVCCKey(r.iter.Item().Key())
}

func (r *rocksDBIterator) Value() []byte {
	val, _ := r.iter.Item().Value()
	return val
}

func (r *rocksDBIterator) ValueProto(msg proto.Message) error {
	val, err := r.iter.Item().Value()
	if err != nil {
		return errors.Trace(err)
	}

	if len(val) <= 0 {
		return nil
	}
	return proto.Unmarshal(val, msg)
}

func (r *rocksDBIterator) unsafeKey() meta.MVCCKey {
	return meta.MVCCKey(r.iter.Item().Key())
}

func (r *rocksDBIterator) unsafeValue() []byte {
	return r.Value()
}

func (r *rocksDBIterator) Error() error {
	//TODO
	return nil
}
