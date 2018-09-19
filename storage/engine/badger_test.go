package engine

import (
	"fmt"
	"os"
	"testing"

	"github.com/taorenhai/ancestor/util/stop"
)

func TestRocksDB(t *testing.T) {
	// Initialize variables.
	key1 := []byte("a")
	key2 := []byte("b")
	key3 := []byte("c")
	val1 := []byte("value")
	val2 := []byte("foo")

	// Set database directory.
	loc := fmt.Sprintf("/tmp/rocksdb_test")

	if _, err := os.Stat(loc); os.IsNotExist(err) {
		t.Log("error: loc is not exist!")
	}

	// Initialize the cache size and create new stopper object.
	const cacheSize = 128 << 20 // 128 MB
	stopper := stop.NewStopper()
	defer stopper.Stop()

	// Create a new RocksDB and open db.
	db := NewInMem(cacheSize, stopper).RocksDB
	if err := db.Open(); err != nil {
		t.Errorf("could not create new rocksdb db instance: %v", err)
		return
	}

	// Put three KeyValue pairs.
	if err := db.Put(key1, val1); err != nil {
		t.Error("Put error!")
	}
	if err := db.Put(key2, []byte("value")); err != nil {
		t.Error("Put error!")
	}
	if err := db.Put(key3, val2); err != nil {
		t.Error("Put error!")
	}

	// Get the values use the key "a".
	if val, err := db.Get(key1); err != nil || val == nil {
		t.Logf("error, got %q: %s\n", val, err)
	} else {
		t.Logf("%s\n", string(val))
	}

	// Check keys between "b" and "\xff".
	kvs, err := Scan(db, key2, []byte("\xff"), 0)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%v\n", kvs)
	}

	// Make a snapshot.
	snap := db.NewSnapshot()
	defer snap.Close()

	if err := db.Put(key2, []byte("value_new")); err != nil {
		t.Error("Put error!")
	}

	if val, err := snap.Get(key2); err != nil || val == nil {
		t.Errorf("error, got %q: %s\n", val, err)
	} else {
		t.Logf("%s\n", string(val))
	}

	// Test batch options
	b := db.NewBatch()
	defer b.Close()

	if err := b.Put(key1, []byte("foo")); err != nil {
		t.Error(err)
	}

	if val, err := b.Get(key1); err != nil {
		t.Error(err)
	} else {
		t.Log(string(val))
	}

	if val, err := db.Get(key1); err != nil {
		t.Error(err)
	} else {
		t.Log(string(val))
	}

	// Commit batch and verify direct engine scan yields correct values.
	if err := b.Commit(); err != nil {
		t.Error(err)
	}

	if val, err := db.Get(key1); err != nil {
		t.Error(err)
	} else {
		t.Log(string(val))
	}
}
