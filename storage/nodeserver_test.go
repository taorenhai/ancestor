package storage

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/client"
	"github.com/taorenhai/ancestor/meta"
)

const (
	etcdHost = "127.0.0.1:2379;127.0.0.1:22379;127.0.0.1:32379"
)

var s client.Storage

func init() {
	var err error
	s, err = client.Open(etcdHost)
	if err != nil {
		log.Fatalf("client open failure, err:%v", err)
	}
}

func TestBeginTransaction(t *testing.T) {
	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transaction failure, err:%v", err)
	}
	defer txn.Commit()

	t.Log("begin transaction success")
}

func TestBeginAndEndTransaction(t *testing.T) {
	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transaction failure, err:%v", err)
		return
	}

	t.Log("begin transacton succes")
	err = txn.Commit()
	if err != nil {
		t.Fatalf("end transaction failure, err:%v", err)
		return
	}

	t.Log("end transaction success")
}

func TestPutData(t *testing.T) {
	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transaction failure, err:%v", err)
		return
	}

	key1 := []byte("test_k1")
	val1 := []byte("value_1_6")
	key2 := []byte("test_k2")
	val2 := []byte("value_2")
	key3 := []byte("test_k3")
	val3 := []byte("value_3")
	key4 := []byte("test_k4")
	val4 := []byte("value_4")

	err = txn.Set(key1, val1)
	if err != nil {
		t.Fatalf("set value failure, key:%v err:%v", string(key1), err)
		return
	}

	t.Logf("set %v success", string(key1))

	err = txn.Set(key2, val2)
	if err != nil {
		t.Fatalf("set value failure, key:%v err:%v", string(key2), err)
		return
	}

	t.Logf("set %v success", string(key2))

	err = txn.Set(key3, val3)
	if err != nil {
		t.Fatalf("set value failure, key:%v err:%v", string(key3), err)
		return
	}

	t.Logf("set %v success", string(key3))

	err = txn.Set(key4, val4)
	if err != nil {
		t.Fatalf("set value failure, key:%v err:%v", string(key4), err)
		return
	}

	t.Logf("set %v success", string(key4))

	err = txn.Commit()
	if err != nil {
		t.Fatalf("end transaction failure, err:%v", err)
		return
	}

}

func TestPutDataMultiTxn(t *testing.T) {

	for i := 0; i < 5; i++ {
		txn, err := s.Begin()
		if err != nil {
			t.Fatalf("begin transaction failure, err:%v", err)
			return
		}

		key1 := []byte(fmt.Sprintf("test_k1"))
		val1 := []byte(fmt.Sprintf("value_1_%v", i+1))

		err = txn.Set(key1, val1)
		if err != nil {
			t.Fatalf("set value failure, key:%v err:%v", string(key1), err)
			return
		}
		txn.Commit()
		t.Logf("set %v success", string(key1))
	}
}

func TestPutDataConflic(t *testing.T) {
	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transaction failure, err:%v", err)
		return
	}
	defer txn.Commit()

	key1 := []byte("test_1")
	val1 := []byte("test_value_2")

	err = txn.Set(key1, val1)
	if err != nil {
		t.Fatalf("set value failure, key:%v err:%v", string(key1), err)
		return
	}
	//txn.EndTransaction(true)
	t.Logf("set %v success", string(key1))
}

func TestGetData(t *testing.T) {
	key1 := []byte("test_k1")
	val1 := []byte("value_1_5")

	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transaction failure, err:%v", err)
		return
	}
	defer txn.Commit()

	val, err := txn.Get(key1)
	if err != nil {
		t.Fatalf("get key value failure, err:%v", err)
		return
	}

	t.Logf("get data key:%v val:%v", string(key1), string(val))

	if bytes.Compare(val, val1) != 0 {
		t.Fatalf("get key value failure, get:%v want:%v err:%v", string(val), string(val1), err)
		return
	}

	t.Logf("get key value success key:%v value:%v", string(key1), string(val))

}

func TestDelData(t *testing.T) {
	key1 := []byte("test_k2")

	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transacion failure, err:%v", err)
		return
	}
	defer txn.Commit()

	t.Logf("get data key:%v", string(key1))

	if err := txn.Delete(key1); err != nil {
		t.Fatalf("delete key value failure, key1:%v err:%v", string(key1), err)
		return
	}

	t.Logf("delete key value success key:%s", string(key1))
}

func TestScanData(t *testing.T) {
	key1 := []byte("mDDLJobHi")

	txn, err := s.Begin()
	if err != nil {
		t.Fatalf("begin transacion failure, err:%v", err)
		return
	}
	defer txn.Commit()

	t.Logf("get data key:%v", string(key1))

	iter, err := txn.Seek(key1)
	if err != nil {
		t.Fatalf("seek data failure, err:%v", err)
		return
	}

	for ; iter.Valid(); iter.Next() {
		t.Logf("seek result key:%v value:%v", string(iter.Key()), string(iter.Value()))
	}

	t.Logf("scan key success")
}

//Test_GetDataConflic get data when other txn write data
func TestGetDataConflic(t *testing.T) {
	log.SetLevel(log.LOG_LEVEL_ALL)

	key := meta.Key(time.Now().Format("test_key.2006-01-02 15:04:05"))
	val := []byte(time.Now().Format("test_val.2006-01-02 15:04:05"))
	t.Logf("key:%q, val:%q", key, val)

	//first start write txn
	txnWrite, err := s.Begin()
	if err != nil {
		t.Fatalf("begin write transacion failure, err:%v", err)
		return
	}
	defer txnWrite.Commit()

	//second start read txn
	txnRead, err := s.Begin()
	if err != nil {
		t.Fatalf("begin read transacion failure, err:%v", err)
		return
	}
	defer txnRead.Rollback()

	// write txn set data
	err = txnWrite.Set(key, val)
	if err != nil {
		t.Fatalf("writeTxn write data failure, err:%v, key:%q", err, key)
		return
	}

	// read txn read data
	result, err := txnRead.Get(key)
	if err == nil || errors.Cause(err) != meta.NotExistError {
		if err == nil {
			t.Fatalf("readTxn get data failure, expect %s, find data:%q", meta.NotExistError.String(), result)
		}
		t.Fatalf("readTxn get data failure, expect %s, err:%v", meta.NotExistError.String(), err)
		return
	}

	// write txn Commit
	err = txnWrite.Commit()
	if err != nil {
		t.Fatalf("writeTxn commit data failure, err:%v, key:%q", err, key)
		return
	}

	// read txn read data
	result, err = txnRead.Get(key)
	if err == nil || errors.Cause(err) != meta.NotExistError {
		if err == nil {
			t.Fatalf("readTxn get data failure, expect %s, find data:%q", meta.NotExistError.String(), result)
		}
		t.Fatalf("readTxn get data failure, expect %s, err:%v", meta.NotExistError.String(), err)
		return
	}

	txnRead.Rollback()

	//start new txn
	txnRead, err = s.Begin()
	if err != nil {
		t.Fatalf("begin new read transacion failure, err:%v", err)
		return
	}

	result, err = txnRead.Get(key)
	if err != nil {
		t.Fatalf("new readTxn get data failure, error:%s", err.Error())
		return
	}

	t.Logf("result:%q", result)
}

// TestPutDataConflic test put data confict
func TestPutDataMultiTxnConflict(t *testing.T) {

	key := meta.Key("test_key1")
	val1 := []byte("test_value_1")
	val2 := []byte("test_value_2")

	t.Logf("key:%q, val1:%q val2:%q", key, val1, val2)

	txnWrite1, err := s.Begin()
	if err != nil {
		t.Fatalf("begin write1 transacion failure, err:%v", err)
		return
	}

	txnWrite2, err := s.Begin()
	if err != nil {
		t.Fatalf("begin write2 transacion failure, err:%v", err)
		return
	}

	// write1 txn set data
	if err = txnWrite1.Set(key, val1); err != nil {
		t.Fatalf("writeTxn1 write data failure, err:%v, key:%q", err, key)
		return
	}

	data, err := txnWrite1.Get(key)
	if err != nil {
		t.Fatalf("writeTxn1 get data failure, err:%v, key:%q", err, key)
		return
	}

	if bytes.Compare(data, val1) != 0 {
		t.Fatalf("writeTxn1 get data failure, value:%q want:%q", data, val1)
		return
	}

	if err = txnWrite2.Set(key, val2); err != nil {
		t.Fatalf("writeTxn2 write data failure, err:%v, key:%q", err, key)
		return
	}

	data, err = txnWrite2.Get(key)
	if err != nil {
		t.Fatalf("writeTxn2 get data failure, err:%v, key:%q", err, key)
		return
	}

	if bytes.Compare(data, val2) != 0 {
		t.Fatalf("writeTxn2 get data failure, value:%q want:%q", data, val2)
		return
	}

	if err = txnWrite1.Commit(); err != nil {
		t.Fatalf("txnWrite1 commit failure")
		return
	}

	if err = txnWrite2.Commit(); err == nil {
		t.Fatalf("txnWrite2 commit success, but we want obtain a error")
		return
	}

	data, err = txnWrite1.Get(key)
	if err != nil && !client.IsErrNotFound(err) {
		t.Fatalf("writeTxn1 get data failure, err:%v, key:%q", err, key)
		return
	}

	data, err = txnWrite2.Get(key)
	if err != nil {
		t.Fatalf("writeTxn2 get data failure, err:%v, key:%q", err, key)
		return
	}

	if bytes.Compare(data, val2) != 0 {
		t.Fatalf("writeTxn2 get data failure, value:%q want:%q", data, val2)
		return
	}
}
