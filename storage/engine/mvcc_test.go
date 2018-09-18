package engine

import (
	"fmt"
	"testing"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/stop"
	"github.com/taorenhai/ancestor/util/uuid"
)

const (
	loc = "/tmp/rocksdb_test"
)

var stopper = stop.NewStopper()

func NewEngine() *RocksDB {
	// Initialize the cache size and create new stopper object.
	const cacheSize = 128 << 20 // 128 MB
	// Create a new RocksDB and open db.
	db := NewRocksDB(loc, cacheSize, stopper)
	if err := db.Open(); err != nil {
		return nil
	}
	return db
}

func TestMVCCPutMeta(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("get new engine failure")
	}
	defer eng.Close()

	for i := 0; i < 10; i++ {
		key := meta.Key(fmt.Sprintf("test_key_%d", i))
		ID := []byte(fmt.Sprintf("txn_id_%d", i))
		txn := &meta.Transaction{}
		txn.ID = ID
		txn.Timestamp = meta.Timestamp{WallTime: 14000 + int64(i*10)}

		err := MVCCPutMeta(eng, key, txn, false)
		if err != nil {
			t.Fatalf("MVCCPutMeta failure, err:%v", err)
		}
	}

}

func TestMVCCPutTxn(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("ge new engine failure")
	}
	defer eng.Close()

	for i := 0; i < 10; i++ {
		ID := []byte(fmt.Sprintf("txn_id_%d", i))
		heartTimestamp := meta.Timestamp{WallTime: 14000 + int64(i*10)}
		txn := &meta.Transaction{}
		txn.ID = ID
		txn.Timestamp = heartTimestamp
		txn.LastHeartbeat = &heartTimestamp.WallTime

		err := MVCCPutTxn(eng, txn, meta.TxnChangeTypeBegin)
		if err != nil {
			t.Fatalf("MVCCPutTxn failure, err:%v", err)
		}
	}

}

func TestMVCCPutData(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("get new engine failure")
	}
	defer eng.Close()

	for i := 0; i < 10; i++ {
		key := meta.Key(fmt.Sprintf("test_key_%d", i))
		timestamp := meta.Timestamp{WallTime: 14000 + int64(i*10)}
		txn := &meta.Transaction{}
		txn.ID = uuid.NewUUID4()
		txn.Timestamp = timestamp
		value := meta.Value{}
		value.Tag = meta.ValueType_BYTES
		value.Bytes = []byte(fmt.Sprintf("test_key_%v", timestamp.WallTime))

		err := MVCCPut(eng, key, txn, value, nil)
		if err != nil {
			t.Fatalf("MVCCPut failure, err:%v", err)
		}
	}

}

func TestMVCCGetMeta(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("get new engine failure")
	}
	defer eng.Close()

	for i := 0; i < 10; i++ {
		key := meta.Key(fmt.Sprintf("test_key_%d", i))
		metaData, err := MVCCGetMeta(eng, key)
		if err != nil {
			t.Fatalf("get metaData failure, err:%v", err)
		}

		t.Logf("metaData:%v", metaData.String())
	}
}

func Test_MVCCGetTxn(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("get new engine failure")
	}
	defer eng.Close()

	key := meta.Key("txn_id_0")
	txn, err := MVCCGetTxn(eng, key)
	if err != nil {
		t.Fatalf("get txn failure, err:%v", err)
	}

	t.Logf("txn:%v", txn.String())
}

func TestMVCCGet(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("get new engine failure")
	}
	defer eng.Close()

	key := meta.Key("test_key_9")
	timestamp := meta.Timestamp{WallTime: 14090}

	value, err := MVCCGet(eng, key, timestamp)
	if err != nil {
		t.Fatalf("MVCCPut failure, err:%v", err)
	}
	t.Logf("key:%v value:%v", string(key), value.String())

	value, err = MVCCGet(eng, key, timestamp)
	if err != nil {
		t.Fatalf("MVCCPut failure, err:%v", err)
	}
	t.Logf("key:%v value:%v", string(key), value.String())

	value, err = MVCCGet(eng, key, timestamp)
	if err != nil {
		t.Fatalf("MVCCPut failure, err:%v", err)
	}
	t.Logf("key:%v value:%v", string(key), value.String())

	value, err = MVCCGet(eng, key, timestamp)
	if err != nil {
		t.Fatalf("MVCCPut failure, err:%v", err)
	}
	t.Logf("key:%v value:%v", string(key), value.String())

}

func TestMVCCResolveWriteIntent(t *testing.T) {
	eng := NewEngine()
	if eng == nil {
		t.Fatalf("get new engine failure")
	}
	defer eng.Close()

	intents := []meta.Intent{}

	for i := 0; i < 3; i++ {
		key := meta.Key(fmt.Sprintf("test_key_%d", i))
		ID := []byte(fmt.Sprintf("txn_id_%d", i))
		txn := &meta.Transaction{}
		txn.ID = ID
		txn.Timestamp = meta.Timestamp{WallTime: 14000 + int64(i*10)}
		if i == 0 {
			txn.Timestamp.WallTime = 140000
		}
		txn.Status = meta.TransactionStatus(i)
		intent := meta.Intent{
			Span: meta.Span{Key: key},
			Txn:  *txn,
		}
		intents = append(intents, intent)
	}

	if err := MVCCResolveWriteIntent(eng, intents); err != nil {
		t.Fatalf("MVCCResolveWriteIntent failure, err:%v", err)
	}

	// case 1: push timestamp
	metaData, err := MVCCGetMeta(eng, meta.Key("test_key_0"))
	if err != nil {
		t.Fatalf("MVCCGetMeta failure, err:%v", err)
	}

	if metaData.Timestamp.WallTime != 140000 {
		t.Fatalf("Check metaData push timestamp failure, metaData: %v", metaData)
	}

	// case 2: committed
	metaData, err = MVCCGetMeta(eng, meta.Key("test_key_1"))
	if err != nil {
		t.Fatalf("MVCCGetMeta failure, err:%v", err)
	}

	if metaData.TransactionID != nil {
		t.Fatalf("Check metaData failure, metaData: %v", metaData)
	}

	// case 3: aborted
	metaData, err = MVCCGetMeta(eng, meta.Key("test_key_2"))
	if err != nil {
		t.Fatalf("MVCCGetMeta failure, err:%v", err)
	}

	if metaData != nil {
		t.Fatalf("Check metaData failure, metaData: %v", metaData)
	}
}
