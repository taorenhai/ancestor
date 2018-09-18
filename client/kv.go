package client

import (
	"github.com/taorenhai/ancestor/meta"
)

// Retriever is the interface wraps the basic Get and Seek methods.
type Retriever interface {
	// Get gets the value for key k from kv store.
	Get(k meta.Key) ([]byte, error)
	// Seek creates an Iterator positioned on the first entry that k <= entry's key.
	// If such entry is not found, it returns an invalid Iterator with no error.
	// The Iterator must be Closed after use.
	Seek(k meta.Key) (Iterator, error)

	// SeekReverse creates a reversed Iterator positioned on the first entry which key is less than k.
	// The returned iterator will iterate from greater key to smaller key.
	// If k is nil, the returned iterator will be positioned at the last key.
	SeekReverse(k meta.Key) (Iterator, error)
}

// Mutator is the interface wraps the basic Set and Delete methods.
type Mutator interface {
	// Set sets the value for key k as v into kv store.
	// v must NOT be nil or empty, otherwise it returns ErrCannotSetNilValue.
	Set(k meta.Key, v []byte) error
	// Delete removes the entry for key k from kv store.
	Delete(k meta.Key) error
}

// Transaction defines the interface for operations inside a Transaction.
type Transaction interface {
	// Retriever is the interface wraps the basic Get and Seek methods.
	Retriever
	// Mutator is the interface wraps the basic Set and Delete methods.
	Mutator

	// Commit commits the transaction operations to KV store.
	Commit() error
	// Rollback undoes the transaction operations to KV store.
	Rollback() error
	// String implements fmt.Stringer interface.
	String() string
	// IsReadOnly checks if the transaction has only performed read operations.
	IsReadOnly() bool
	// StartTS returns the transaction start timestamp.
	StartTS() uint64
}

// Snapshot defines the interface for the snapshot fetched from KV store.
type Snapshot interface {
	// Retriever is the interface wraps the basic Get and Seek methods.
	Retriever
	// BatchGet gets a batch of values from snapshot.
	BatchGet(keys []meta.Key) (map[string][]byte, error)
	// Release releases the snapshot to store.
	Release()
}

// Storage defines the interface for storage.
type Storage interface {
	// Begin transaction
	Begin(priority ...int) (Transaction, error)
	// GetSnapshot gets a snapshot that is able to read any data which data is <= ver.
	// if ver is MaxVersion or > current max committed version, we will use current version for this snapshot.
	GetSnapshot(ver meta.Timestamp) (Snapshot, error)
	// Close store
	Close() error
	// Storage's unique ID
	UUID() string
	// CurrentVersion returns current max committed version.
	CurrentVersion() (meta.Timestamp, error)
	// GetAdmin return a new admin interface for pd, node only.
	GetAdmin() Admin
}

// Iterator is the interface for a iterator on KV store.
type Iterator interface {
	Valid() bool
	Key() meta.Key
	Value() []byte
	Next() error
	Close()
}

// Admin defines the interface for node, pd.
type Admin interface {
	// GetNewID get new ID from pd.
	GetNewID() (meta.RangeID, error)
	// GetRangeDescriptor get a rangeDescriptor from pd.
	GetRangeDescriptor(key meta.Key) (meta.RangeDescriptor, error)
	// SetLocalServer set current node id for local router.
	SetLocalServer(id meta.NodeID, nodeServer meta.Executer)
	// SetRangeStats set range stats to pd.
	SetRangeStats(stats []meta.RangeStatsInfo) error
	// SetNodeStats set node stats to pd.
	SetNodeStats(stats meta.NodeStatsInfo) error

	// GetNodes get nodeDescriptor from pd.
	GetNodes() ([]meta.NodeDescriptor, error)
	// SetNodes set nodeDescriptor to pd.
	SetNodes(nds []meta.NodeDescriptor) error

	// GetRangeDescriptors get RangeDescriptor from pd.
	GetRangeDescriptors(keys ...meta.Key) ([]meta.RangeDescriptor, error)
	// SetRangeDescriptors set RangeDescriptor to pd.
	SetRangeDescriptors(rds []meta.RangeDescriptor) error

	// GetTimestamp get timestamp from pd.
	GetTimestamp() (meta.Timestamp, error)

	// ResolveIntent send ResolveIntentRequest to node.
	ResolveIntent(key meta.Key, txn *meta.Transaction) error
	// PushTxn send PushTransactionRequest to node.
	PushTxn(pushType meta.PushTxnType, key meta.Key, txn *meta.Transaction) (*meta.Transaction, error)

	// Bootstrap
	Bootstrap(address string, capacity int64) (meta.NodeDescriptor, error)

	// CreateReplica
	CreateReplica(id meta.NodeID, rd meta.RangeDescriptor) error
}
