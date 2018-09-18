package engine

import (
	"github.com/taorenhai/ancestor/util/stop"
)

// InMem wraps RocksDB and configures it for in-memory only storage.
type InMem struct {
	*RocksDB
}

// NewInMem allocates and returns a new, opened InMem engine.
func NewInMem(cacheSize int64, stopper *stop.Stopper) InMem {
	db := InMem{
		RocksDB: newMemRocksDB(cacheSize, stopper),
	}
	if err := db.Open(); err != nil {
		panic(err)
	}
	return db
}
