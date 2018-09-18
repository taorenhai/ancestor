package main_test

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zssky/log"
	//	. "github.com/pingcap/check"

	"github.com/taorenhai/ancestor/client"
)

const (
	etcdHosts  = "127.0.0.1:2379;127.0.0.1:22379;127.0.0.1:32379"
	startIndex = 0
	testCount  = 2
	indexStep  = 2
)

type testKVSuite struct {
	s client.Storage
}

var _ = Suite(&testKVSuite{})

func insertData(c *C, txn client.Transaction) {
	for i := startIndex; i < testCount; i++ {
		val := encodeInt(i * indexStep)
		err := txn.Set(val, val)
		c.Assert(err, IsNil)
	}
}

func mustDel(c *C, txn client.Transaction) {
	for i := startIndex; i < testCount; i++ {
		val := encodeInt(i * indexStep)
		err := txn.Delete(val)
		c.Assert(err, IsNil)
	}
}

func encodeInt(n int) []byte {
	return []byte(fmt.Sprintf("%010d", n))
}

func decodeInt(s []byte) int {
	var n int
	fmt.Sscanf(string(s), "%010d", &n)
	return n
}

func valToStr(c *C, iter client.Iterator) string {
	val := iter.Value()
	return string(val)
}

func checkSeek(c *C, txn client.Transaction) {
	for i := startIndex; i < testCount; i++ {
		val := encodeInt(i * indexStep)
		iter, err := txn.Seek(val)
		c.Assert(err, IsNil)
		c.Assert([]byte(iter.Key()), BytesEquals, val)
		c.Assert(decodeInt([]byte(valToStr(c, iter))), Equals, i*indexStep)
		iter.Close()
	}

	// Test iterator Next()
	for i := startIndex; i < testCount-1; i++ {
		val := encodeInt(i * indexStep)
		iter, err := txn.Seek(val)
		c.Assert(err, IsNil)
		c.Assert([]byte(iter.Key()), BytesEquals, val)
		c.Assert(valToStr(c, iter), Equals, string(val))

		err = iter.Next()
		c.Assert(err, IsNil)
		c.Assert(iter.Valid(), IsTrue)

		val = encodeInt((i + 1) * indexStep)
		c.Assert([]byte(iter.Key()), BytesEquals, val)
		c.Assert(valToStr(c, iter), Equals, string(val))
		iter.Close()
	}

	// Non exist and beyond maximum seek test
	iter, err := txn.Seek(encodeInt(testCount * indexStep))
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsFalse)

	// Non exist but between existing keys seek test,
	// it returns the smallest key that larger than the one we are seeking
	inBetween := encodeInt((testCount-1)*indexStep - 1)
	last := encodeInt((testCount - 1) * indexStep)
	iter, err = txn.Seek(inBetween)
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)
	c.Assert([]byte(iter.Key()), Not(BytesEquals), inBetween)
	c.Assert([]byte(iter.Key()), BytesEquals, last)
	iter.Close()
}

func mustNotGet(c *C, txn client.Transaction) {
	for i := startIndex; i < testCount; i++ {
		s := encodeInt(i * indexStep)
		_, err := txn.Get(s)
		c.Assert(err, NotNil)
	}
}

func mustGet(c *C, txn client.Transaction) {
	for i := startIndex; i < testCount; i++ {
		s := encodeInt(i * indexStep)
		val, err := txn.Get(s)
		c.Assert(err, IsNil)
		c.Assert(string(val), Equals, string(s))
	}
}

func deleteAll(c *C, txn client.Transaction) {
	it, err := txn.Seek([]byte("\x00"))
	c.Assert(err, IsNil)
	for it.Valid() {
		log.Debugf("delete key:%q", it.Key())
		err = txn.Delete([]byte(it.Key()))
		c.Assert(err, IsNil)
		err = it.Next()
		c.Assert(err, IsNil)
	}
}

func (s *testKVSuite) SetUpSuite(c *C) {
	store, err := client.Open(etcdHosts)
	c.Assert(err, IsNil)
	s.s = store
}

func (s *testKVSuite) TearDownSuite(c *C) {
	err := s.s.Close()
	c.Assert(err, IsNil)
}

func TestT(t *testing.T) {
	TestingT(t)
}

//-------------------------Basic Test Case-------------------------------

func (s *testKVSuite) TestGetSet(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	insertData(c, txn)

	mustGet(c, txn)

	// Check transaction results
	err = txn.Commit()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	mustGet(c, txn)
	mustDel(c, txn)
}

func (s *testKVSuite) TestSeek(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	deleteAll(c, txn)
	insertData(c, txn)
	checkSeek(c, txn)

	// Check transaction results
	err = txn.Commit()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	checkSeek(c, txn)
	mustDel(c, txn)
}

func (s *testKVSuite) TestBasicSeek(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	deleteAll(c, txn)

	txn.Set([]byte("1"), []byte("1"))
	txn.Commit()
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	it, err := txn.Seek([]byte("2"))
	c.Assert(err, IsNil)
	c.Assert(it.Valid(), Equals, false)
	txn.Delete([]byte("1"))
}

func (s *testKVSuite) TestSeekMin(c *C) {
	kvs := []struct {
		key   string
		value string
	}{
		{"DATA_test_main_db_tbl_tbl_test_record__00000000000000000001", "lock-version"},
		{"DATA_test_main_db_tbl_tbl_test_record__00000000000000000001_0002", "1"},
		{"DATA_test_main_db_tbl_tbl_test_record__00000000000000000001_0003", "hello"},
		{"DATA_test_main_db_tbl_tbl_test_record__00000000000000000002", "lock-version"},
		{"DATA_test_main_db_tbl_tbl_test_record__00000000000000000002_0002", "2"},
		{"DATA_test_main_db_tbl_tbl_test_record__00000000000000000002_0003", "hello"},
	}

	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	for _, kv := range kvs {
		txn.Set([]byte(kv.key), []byte(kv.value))
	}

	it, err := txn.Seek([]byte("\x00"))
	for it.Valid() {
		fmt.Printf("%s, %s\n", it.Key(), it.Value())
		it.Next()
	}

	it, err = txn.Seek([]byte("DATA_test_main_db_tbl_tbl_test_record__00000000000000000000"))
	c.Assert(err, IsNil)
	c.Assert(string(it.Key()), Equals, "DATA_test_main_db_tbl_tbl_test_record__00000000000000000001")

	for _, kv := range kvs {
		txn.Delete([]byte(kv.key))
	}
}

func (s *testKVSuite) TestInc(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	key := []byte("incKey")
	n, err := client.IncInt64(txn, key, 100)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(100))

	// Check transaction results
	err = txn.Commit()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	n, err = client.IncInt64(txn, key, -200)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(-100))

	err = txn.Delete(key)
	c.Assert(err, IsNil)

	n, err = client.IncInt64(txn, key, 100)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(100))

	err = txn.Delete(key)
	c.Assert(err, IsNil)

	err = txn.Commit()
	c.Assert(err, IsNil)
}

func (s *testKVSuite) TestSetNil(c *C) {
	txn, err := s.s.Begin()
	defer txn.Commit()
	c.Assert(err, IsNil)
	err = txn.Set([]byte("1"), nil)
	c.Assert(err, NotNil)
}

func (s *testKVSuite) TestDelete(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	insertData(c, txn)

	mustDel(c, txn)

	mustNotGet(c, txn)
	txn.Commit()

	// Try get
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	mustNotGet(c, txn)

	// Insert again
	insertData(c, txn)
	txn.Commit()

	// Delete all
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	mustDel(c, txn)
	txn.Commit()

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	mustNotGet(c, txn)
	txn.Commit()
}

func (s *testKVSuite) TestDelete2(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	val := []byte("test")
	txn.Set([]byte("DATA_test_tbl_department_record__0000000001_0003"), val)
	txn.Set([]byte("DATA_test_tbl_department_record__0000000001_0004"), val)
	txn.Set([]byte("DATA_test_tbl_department_record__0000000002_0003"), val)
	txn.Set([]byte("DATA_test_tbl_department_record__0000000002_0004"), val)
	txn.Commit()

	// Delete all
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	it, err := txn.Seek([]byte("DATA_test_tbl_department_record__0000000001_0003"))
	c.Assert(err, IsNil)
	for it.Valid() {
		err = txn.Delete([]byte(it.Key()))
		c.Assert(err, IsNil)
		err = it.Next()
		c.Assert(err, IsNil)
	}
	txn.Commit()

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	it, _ = txn.Seek([]byte("DATA_test_tbl_department_record__000000000"))
	c.Assert(it.Valid(), IsFalse)
	txn.Commit()

}

func (s *testKVSuite) TestRollback(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn.Rollback()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	insertData(c, txn)

	mustGet(c, txn)

	err = txn.Rollback()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	for i := startIndex; i < testCount; i++ {
		_, err1 := txn.Get([]byte(strconv.Itoa(i)))
		c.Assert(err1, NotNil)
	}

	err = txn.Commit()
	c.Assert(err, IsNil)

	// case 2
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	insertData(c, txn)

	mustGet(c, txn)

	err = txn.Commit()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	mustDel(c, txn)

	err = txn.Rollback()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	mustGet(c, txn)
	mustDel(c, txn)
}

func (s *testKVSuite) TestDBClose(c *C) {
	store, err := client.Open(etcdHosts)
	c.Assert(err, IsNil)

	txn, err := store.Begin()
	c.Assert(err, IsNil)

	err = txn.Set([]byte("a"), []byte("b"))
	c.Assert(err, IsNil)

	err = txn.Commit()
	c.Assert(err, IsNil)

	_, err = store.CurrentVersion()
	c.Assert(err, IsNil)

	snap, err := store.GetSnapshot(client.MaxVersion)
	c.Assert(err, IsNil)

	_, err = snap.Get([]byte("a"))
	c.Assert(err, IsNil)

	txn, err = store.Begin()
	c.Assert(err, IsNil)

	err = store.Close()
	c.Assert(err, IsNil)

	_, err = store.Begin()
	c.Assert(err, NotNil)

	_, err = store.GetSnapshot(client.MaxVersion)
	c.Assert(err, NotNil)

	err = txn.Set([]byte("a"), []byte("b"))
	c.Assert(err, IsNil)

	err = txn.Commit()
	c.Assert(err, NotNil)

	snap.Release()

	// re-open store
	store, err = client.Open(etcdHosts)
	c.Assert(err, IsNil)

	txn, err = store.Begin()
	c.Assert(err, IsNil)

	err = txn.Commit()
	c.Assert(err, IsNil)

	err = store.Close()
	c.Assert(err, IsNil)
}

func (s *testKVSuite) TestDBDeadlock(c *C) {
	client.RunInNewTxn(s.s, false, func(txn client.Transaction) error {
		txn.Set([]byte("a"), []byte("0"))
		client.IncInt64(txn, []byte("a"), 1)

		client.RunInNewTxn(s.s, false, func(txn client.Transaction) error {
			txn.Set([]byte("b"), []byte("0"))
			client.IncInt64(txn, []byte("b"), 1)

			return nil
		})

		return nil
	})

	client.RunInNewTxn(s.s, false, func(txn client.Transaction) error {
		n, err := client.GetInt64(txn, []byte("a"))
		c.Assert(err, IsNil)
		c.Assert(n, Equals, int64(1))

		n, err = client.GetInt64(txn, []byte("b"))
		c.Assert(err, IsNil)
		c.Assert(n, Equals, int64(1))
		return nil
	})
}

//-----------------Basic Test Case With Multi Range---------------

func (s *testKVSuite) TestGetSetMultiRange(c *C) {
	kv1 := []byte("\x00\x00_0")
	kv2 := []byte("\xf0\xf0_1")

	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn.Set(kv1, kv1)
	c.Assert(err, IsNil)

	err = txn.Set(kv2, kv2)
	c.Assert(err, IsNil)

	val, err := txn.Get(kv1)
	c.Assert(err, IsNil)
	c.Assert(val, BytesEquals, kv1)

	val, err = txn.Get(kv2)
	c.Assert(err, IsNil)
	c.Assert(val, BytesEquals, kv2)

	// Check transaction results
	err = txn.Commit()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	val, err = txn.Get(kv1)
	c.Assert(err, IsNil)
	c.Assert(val, BytesEquals, kv1)

	val, err = txn.Get(kv2)
	c.Assert(err, IsNil)
	c.Assert(val, BytesEquals, kv2)

	// Clear
	err = txn.Delete(kv1)
	c.Assert(err, IsNil)

	err = txn.Delete(kv2)
	c.Assert(err, IsNil)
}

func (s *testKVSuite) TestSeekMultiRange(c *C) {
	kv1 := []byte("\x00\x00_0")
	kv2 := []byte("\xf0\xf0_1")

	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn.Set(kv1, kv1)
	c.Assert(err, IsNil)

	err = txn.Set(kv2, kv2)
	c.Assert(err, IsNil)

	iter, err := txn.Seek(kv1)
	c.Assert(err, IsNil)
	c.Assert([]byte(iter.Key()), BytesEquals, kv1)
	c.Assert(valToStr(c, iter), Equals, string(kv1))
	iter.Close()

	iter, err = txn.Seek(kv2)
	c.Assert(err, IsNil)
	c.Assert([]byte(iter.Key()), BytesEquals, kv2)
	c.Assert(valToStr(c, iter), Equals, string(kv2))
	iter.Close()

	// Test iterator Next()
	iter, err = txn.Seek(kv1)
	c.Assert(err, IsNil)
	c.Assert([]byte(iter.Key()), BytesEquals, kv1)
	c.Assert(valToStr(c, iter), Equals, string(kv1))

	err = iter.Next()
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)

	c.Assert([]byte(iter.Key()), BytesEquals, kv2)
	c.Assert(valToStr(c, iter), Equals, string(kv2))
	iter.Close()

	// Non exist and beyond maximum seek test
	iter, err = txn.Seek([]byte("\xf1\xf1"))
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsFalse)

	// Non exist but between existing keys seek test,
	// it returns the smallest key that larger than the one we are seeking
	iter, err = txn.Seek([]byte("\x0f\x0f"))
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)
	c.Assert([]byte(iter.Key()), Not(BytesEquals), []byte("\x0f\x0f"))
	c.Assert([]byte(iter.Key()), BytesEquals, kv2)
	iter.Close()

	// Check transaction results
	err = txn.Commit()
	c.Assert(err, IsNil)

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	iter, err = txn.Seek(kv1)
	c.Assert(err, IsNil)
	c.Assert([]byte(iter.Key()), BytesEquals, kv1)
	c.Assert(valToStr(c, iter), Equals, string(kv1))
	iter.Close()

	iter, err = txn.Seek(kv2)
	c.Assert(err, IsNil)
	c.Assert([]byte(iter.Key()), BytesEquals, kv2)
	c.Assert(valToStr(c, iter), Equals, string(kv2))
	iter.Close()

	// Test iterator Next()
	iter, err = txn.Seek(kv1)
	c.Assert(err, IsNil)
	c.Assert([]byte(iter.Key()), BytesEquals, kv1)
	c.Assert(valToStr(c, iter), Equals, string(kv1))

	err = iter.Next()
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)

	c.Assert([]byte(iter.Key()), BytesEquals, kv2)
	c.Assert(valToStr(c, iter), Equals, string(kv2))
	iter.Close()

	// Non exist and beyond maximum seek test
	iter, err = txn.Seek([]byte("\xf1\xf1"))
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsFalse)

	// Non exist but between existing keys seek test,
	// it returns the smallest key that larger than the one we are seeking
	iter, err = txn.Seek([]byte("\x0f\x0f"))
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)
	c.Assert([]byte(iter.Key()), Not(BytesEquals), []byte("\x0f\x0f"))
	c.Assert([]byte(iter.Key()), BytesEquals, kv2)
	iter.Close()

	txn.Delete(kv1)
	txn.Delete(kv2)
}

func (s *testKVSuite) TestDeleteMultiRange(c *C) {
	kv1 := []byte("\x00\x00_0")
	kv2 := []byte("\xf0\xf0_1")

	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn.Set(kv1, kv1)
	c.Assert(err, IsNil)

	err = txn.Set(kv2, kv2)
	c.Assert(err, IsNil)

	err = txn.Delete(kv1)
	c.Assert(err, IsNil)

	err = txn.Delete(kv2)
	c.Assert(err, IsNil)

	_, err = txn.Get(kv1)
	c.Assert(err, NotNil)

	_, err = txn.Get(kv2)
	c.Assert(err, NotNil)

	txn.Commit()

	// Try get
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	_, err = txn.Get(kv1)
	c.Assert(err, NotNil)

	_, err = txn.Get(kv2)
	c.Assert(err, NotNil)

	// Insert again
	err = txn.Set(kv1, kv1)
	c.Assert(err, IsNil)

	err = txn.Set(kv2, kv2)
	c.Assert(err, IsNil)

	txn.Commit()

	// Delete all
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	err = txn.Delete(kv1)
	c.Assert(err, IsNil)

	err = txn.Delete(kv2)
	c.Assert(err, IsNil)

	txn.Commit()

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	_, err = txn.Get(kv1)
	c.Assert(err, NotNil)

	_, err = txn.Get(kv2)
	c.Assert(err, NotNil)

	txn.Commit()
}

func (s *testKVSuite) TestDelete2MultiRange(c *C) {
	kv1 := []byte("\x00\x00_0000000001_0003")
	kv2 := []byte("\x00\x00_0000000001_0004")
	kv3 := []byte("\xfe\xfe_0000000002_0003")
	kv4 := []byte("\xfe\xfe_0000000002_0004")

	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	txn.Set(kv1, kv1)
	txn.Set(kv2, kv2)
	txn.Set(kv3, kv3)
	txn.Set(kv4, kv4)
	txn.Commit()

	// Delete all
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)

	it, err := txn.Seek([]byte("\x00\x00_0000000001_0003"))
	c.Assert(err, IsNil)
	for it.Valid() {
		err = txn.Delete([]byte(it.Key()))
		c.Assert(err, IsNil)
		err = it.Next()
		c.Assert(err, IsNil)
	}
	txn.Commit()

	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	it, _ = txn.Seek([]byte("\x00\x00_000000000"))
	c.Assert(it.Valid(), IsFalse)
	it, _ = txn.Seek([]byte("\xfe\xfe_000000000"))
	c.Assert(it.Valid(), IsFalse)

	_, err = txn.Get(kv1)
	c.Assert(err, NotNil)

	_, err = txn.Get(kv2)
	c.Assert(err, NotNil)

	_, err = txn.Get(kv3)
	c.Assert(err, NotNil)

	_, err = txn.Get(kv4)
	c.Assert(err, NotNil)

	txn.Commit()

}

//--------------------TiDB Operation Test Case-------------------

func (s *testKVSuite) TestBasicTable(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	for i := 1; i < 5; i++ {
		b := []byte(strconv.Itoa(i))
		txn.Set(b, b)
	}
	txn.Commit()
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	err = txn.Set([]byte("1"), []byte("1"))
	c.Assert(err, IsNil)

	it, err := txn.Seek([]byte("0"))
	c.Assert(err, IsNil)
	c.Assert(string(it.Key()), Equals, "1")

	err = txn.Set([]byte("0"), []byte("0"))
	c.Assert(err, IsNil)
	it, err = txn.Seek([]byte("0"))
	c.Assert(err, IsNil)
	c.Assert(string(it.Key()), Equals, "0")
	err = txn.Delete([]byte("0"))
	c.Assert(err, IsNil)

	txn.Delete([]byte("1"))
	it, err = txn.Seek([]byte("0"))
	c.Assert(err, IsNil)
	c.Assert(string(it.Key()), Equals, "2")

	err = txn.Delete([]byte("3"))
	c.Assert(err, IsNil)
	it, err = txn.Seek([]byte("2"))
	c.Assert(err, IsNil)
	c.Assert(string(it.Key()), Equals, "2")

	it, err = txn.Seek([]byte("3"))
	c.Assert(err, IsNil)
	c.Assert(string(it.Key()), Equals, "4")
	err = txn.Delete([]byte("2"))
	c.Assert(err, IsNil)
	err = txn.Delete([]byte("4"))
	c.Assert(err, IsNil)
}

func (s *testKVSuite) TestConditionIfNotExist(c *C) {
	var success int64
	cnt := 100
	b := []byte("1")
	var wg sync.WaitGroup
	wg.Add(cnt)
	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			txn, err := s.s.Begin()
			c.Assert(err, IsNil)
			err = txn.Set(b, b)
			if err != nil {
				return
			}
			err = txn.Commit()
			if err == nil {
				atomic.AddInt64(&success, 1)
			}
		}()
	}
	wg.Wait()
	// At least one txn can success.
	c.Assert(success, Greater, int64(0))

	// Clean up
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	err = txn.Delete(b)
	c.Assert(err, IsNil)
	err = txn.Commit()
	c.Assert(err, IsNil)
}

func (s *testKVSuite) TestConditionIfEqual(c *C) {
	var success int64
	cnt := 100
	b := []byte("1")
	var wg sync.WaitGroup
	wg.Add(cnt)

	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	txn.Set(b, b)
	err = txn.Commit()
	c.Assert(err, IsNil)

	for i := 0; i < cnt; i++ {
		go func() {
			defer wg.Done()
			// Use txn1/err1 instead of txn/err is
			// to pass `go tool vet -shadow` check.
			txn1, err1 := s.s.Begin()
			c.Assert(err1, IsNil)
			txn1.Set(b, []byte("newValue"))
			err1 = txn1.Commit()
			if err1 == nil {
				atomic.AddInt64(&success, 1)
			}
		}()
	}
	wg.Wait()
	c.Assert(success, Greater, int64(0))

	// Clean up
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	err = txn.Delete(b)
	c.Assert(err, IsNil)
	err = txn.Commit()
	c.Assert(err, IsNil)
}

func (s *testKVSuite) TestConditionUpdate(c *C) {
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	txn.Delete([]byte("b"))
	client.IncInt64(txn, []byte("a"), 1)
	err = txn.Commit()
	c.Assert(err, IsNil)
}

//------------------------Isolation Test Case------------------------------

// TestIsolationRW test read/write conflict
func (s *testKVSuite) TestIsolationRW(c *C) {
	kv1 := []byte("isolation_rw_test1")
	kv2 := []byte("isolation_rw_test2")
	kv3 := []byte("isolation_rw_test3")

	// case 1: old txn should't get new txn's data.
	txn1, err := s.s.Begin()
	c.Assert(err, IsNil)

	txn2, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn2.Set(kv1, kv1)
	c.Assert(err, IsNil)

	_, err = txn1.Get(kv1)
	c.Assert(err, NotNil)

	err = txn2.Commit()
	c.Assert(err, IsNil)

	_, err = txn1.Get(kv1)
	c.Assert(err, NotNil)

	err = txn1.Set(kv3, kv3)
	c.Assert(err, IsNil)

	err = txn1.Commit()
	c.Assert(err, IsNil)

	// clear test key
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn.Delete(kv1)
	c.Assert(err, IsNil)

	err = txn.Commit()
	c.Assert(err, IsNil)

	// case 2: push timestamp
	txn1, err = s.s.Begin()
	c.Assert(err, IsNil)

	txn2, err = s.s.Begin()
	c.Assert(err, IsNil)

	err = txn1.Set(kv1, kv1)
	c.Assert(err, IsNil)

	err = txn1.Delete(kv3)
	c.Assert(err, IsNil)

	err = txn2.Set(kv2, kv2)
	c.Assert(err, IsNil)

	_, err = txn2.Get(kv1)
	c.Assert(err, NotNil)

	_, err = txn2.Get(kv3)
	c.Assert(err, IsNil)

	val, err := txn2.Get(kv2)
	c.Assert(err, IsNil)
	c.Assert(val, BytesEquals, kv2)

	iter, err := txn2.Seek(kv1)
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)
	c.Assert(string(iter.Key()), Equals, string(kv2))
	c.Assert(valToStr(c, iter), Equals, string(kv2))
	iter.Close()

	err = txn1.Commit()
	c.Assert(err, IsNil)

	_, err = txn2.Get(kv1)
	c.Assert(err, NotNil)

	_, err = txn2.Get(kv3)
	c.Assert(err, IsNil)

	val, err = txn2.Get(kv2)
	c.Assert(err, IsNil)
	c.Assert(string(val), Equals, string(kv2))

	iter, err = txn2.Seek(kv1)
	c.Assert(err, IsNil)
	c.Assert(iter.Valid(), IsTrue)
	c.Assert(string(iter.Key()), Equals, string(kv2))
	c.Assert(valToStr(c, iter), Equals, string(kv2))
	iter.Close()

	err = txn2.Commit()
	c.Assert(err, IsNil)

	// case 3: new txn get old txn's modified data when old txn committed
	txn1, err = s.s.Begin()
	c.Assert(err, IsNil)

	txn2, err = s.s.Begin()
	c.Assert(err, IsNil)

	val1, err := txn2.Get(kv1)
	c.Assert(err, IsNil)

	err = txn1.Set(kv1, []byte("test"))
	c.Assert(err, IsNil)

	err = txn1.Commit()
	c.Assert(err, IsNil)

	val2, err := txn2.Get(kv1)
	c.Assert(err, IsNil)
	c.Assert(string(val2), Equals, string(val1))

	err = txn2.Commit()
	c.Assert(err, IsNil)

	// clear test key
	txn, err = s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	err = txn.Delete(kv1)
	c.Assert(err, IsNil)

	err = txn.Delete(kv2)
	c.Assert(err, IsNil)

	err = txn.Delete(kv3)
	c.Assert(err, IsNil)
}

// TestIsolationRaceCondition test isolation on race condition.
func (s *testKVSuite) TestIsolationRaceCondition(c *C) {
	kv1 := []byte("isolation_race_test_key1")
	kv2 := []byte("isolation_race_test_key2")

	txn1, err := s.s.Begin()
	c.Assert(err, IsNil)

	txn2, err := s.s.Begin()
	c.Assert(err, IsNil)

	txn3, err := s.s.Begin()
	c.Assert(err, IsNil)

	err = txn1.Set(kv1, kv1)
	c.Assert(err, IsNil)

	err = txn2.Set(kv2, kv2)
	c.Assert(err, IsNil)

	_, err = txn1.Get(kv2)
	c.Assert(err, NotNil)

	_, err = txn3.Get(kv1)
	c.Assert(err, NotNil)

	err = txn2.Commit()
	c.Assert(err, IsNil)

	_, err = txn1.Get(kv2)
	c.Assert(err, NotNil)

	err = txn1.Commit()
	c.Assert(err, IsNil)

	err = txn3.Commit()
	c.Assert(err, IsNil)

	// clear test key
	txn, err := s.s.Begin()
	c.Assert(err, IsNil)
	defer txn.Commit()

	err = txn.Delete(kv1)
	c.Assert(err, IsNil)

	err = txn.Delete(kv2)
	c.Assert(err, IsNil)
}

// TestIsolationInc test isolation for increment
func (s *testKVSuite) TestIsolationInc(c *C) {
	threadCnt := 4

	ids := make(map[int64]struct{}, threadCnt*100)
	var m sync.Mutex
	var wg sync.WaitGroup

	wg.Add(threadCnt)
	for i := 0; i < threadCnt; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				var id int64
				err := client.RunInNewTxn(s.s, true, func(txn client.Transaction) error {
					var err1 error
					id, err1 = client.IncInt64(txn, []byte("key"), 1)
					return err1
				})
				c.Assert(err, IsNil)

				m.Lock()
				_, ok := ids[id]
				ids[id] = struct{}{}
				m.Unlock()
				c.Assert(ok, IsFalse)
			}
		}()
	}

	wg.Wait()
}

// TestIsolationMultiInc test isolation for increment use different keys.
func (s *testKVSuite) TestIsolationMultiInc(c *C) {
	threadCnt := 10
	incCnt := 100
	keyCnt := 4

	keys := make([][]byte, 0, keyCnt)
	for i := 0; i < keyCnt; i++ {
		keys = append(keys, []byte(fmt.Sprintf("test_key_%d", i)))
	}

	var wg sync.WaitGroup

	wg.Add(threadCnt)
	for i := 0; i < threadCnt; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incCnt; j++ {
				err1 := client.RunInNewTxn(s.s, true, func(txn client.Transaction) error {
					for _, key := range keys {
						_, err2 := client.IncInt64(txn, key, 1)
						if err2 != nil {
							return err2
						}
					}

					return nil
				})
				c.Assert(err1, IsNil)
			}
		}()
	}

	wg.Wait()

	for i := 0; i < keyCnt; i++ {
		err := client.RunInNewTxn(s.s, false, func(txn client.Transaction) error {
			for _, key := range keys {
				id, err1 := client.GetInt64(txn, key)
				if err1 != nil {
					return err1
				}
				c.Assert(id, Equals, int64(threadCnt*incCnt))
			}
			return nil
		})
		c.Assert(err, IsNil)
	}
}

//--------------------------------------------------------------------------------------

type txnObj struct {
	key     string
	txnName string
}

var (
	txnKeyMap map[string]string
	checkChan chan *txnObj
)

func checkTxnResult() {
	txnKeyMap = make(map[string]string)
	for {
		kv := <-checkChan
		if oldTxnName, ok := txnKeyMap[kv.key]; ok {
			log.Fatalf("old txn:%s write key:%s, current txn:%s", oldTxnName, kv.key, kv.txnName)
		}
		txnKeyMap[kv.key] = kv.txnName
		log.Debugf("%s put new record:%s", kv.txnName, kv.key)
	}
}

// TestIsolationMultiRange test isolation for increment use different keys distribution to different ranges.
func (s *testKVSuite) TestIsolationMultiRange(c *C) {
	checkChan = make(chan *txnObj, 1000)
	go checkTxnResult()

	threadCnt := 4
	incCnt := 100
	keyCnt := 2

	keys := make([][]byte, 0, keyCnt)
	keys = append(keys, []byte("\x00\x00_test_key_1"))
	keys = append(keys, []byte("\xfe\xfe_test_key_2"))

	var wg sync.WaitGroup

	wg.Add(threadCnt)
	for i := 0; i < threadCnt; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incCnt; j++ {
				vals := make([]int64, keyCnt)
				txnName := ""
				var err error
				err = client.RunInNewTxn(s.s, true, func(txn client.Transaction) error {
					for k, key := range keys {
						vals[k], err = client.IncInt64(txn, key, 1)
						if err != nil {
							return err
						}
					}
					txnName = txn.String()

					return nil
				})
				c.Assert(err, IsNil)
				for k, key := range keys {
					checkChan <- &txnObj{key: fmt.Sprintf("%q_%d", key, vals[k]), txnName: txnName}
				}
			}
		}()
	}

	wg.Wait()

	for i := 0; i < keyCnt; i++ {
		err := client.RunInNewTxn(s.s, false, func(txn client.Transaction) error {
			for _, key := range keys {
				id, err1 := client.GetInt64(txn, key)
				if err1 != nil {
					return err1
				}
				c.Assert(id, Equals, int64(threadCnt*incCnt))
			}
			return nil
		})
		c.Assert(err, IsNil)
	}
}
