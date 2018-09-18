#include <algorithm>
#include <limits>
#include <google/protobuf/repeated_field.h>
#include "rocksdb/cache.h"
#include "rocksdb/compaction_filter.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/merge_operator.h"
#include "rocksdb/options.h"
#include "rocksdb/table.h"
#include "rocksdb/utilities/write_batch_with_index.h"

#include "db.h"

extern "C" {
#include "_cgo_export.h"

struct DBBatch {
  int updates;
  rocksdb::WriteBatchWithIndex rep;

  DBBatch()
      : updates(0) {
  }
};

struct DBEngine {
  rocksdb::DB* rep;
  rocksdb::Env* memenv;
};

struct DBIterator {
  rocksdb::Iterator* rep;
};

struct DBSnapshot {
  rocksdb::DB* db;
  const rocksdb::Snapshot* rep;
};

}  // extern "C"

namespace {

// NOTE: these constants must be kept in sync with the values in
// storage/engine/keys.go. Both kKeyLocalRangeIDPrefix and
// kKeyLocalRangePrefix are the mvcc-encoded prefixes.
const int kKeyLocalRangePrefixSize = 4;
const rocksdb::Slice kKeyLocalRangeIDPrefix("\x31\x00\xff\x00\xff\x00\xffi", 8);
const rocksdb::Slice kKeyLocalRangePrefix("\x31\x00\xff\x00\xff\x00\xffk", 8);
const rocksdb::Slice kKeyLocalTransactionSuffix("\x00\x01txn-", 6);

const DBStatus kSuccess = { NULL, 0 };

std::string ToString(DBSlice s) {
  return std::string(s.data, s.len);
}

rocksdb::Slice ToSlice(DBSlice s) {
  return rocksdb::Slice(s.data, s.len);
}

rocksdb::Slice ToSlice(DBString s) {
  return rocksdb::Slice(s.data, s.len);
}

DBSlice ToDBSlice(const rocksdb::Slice& s) {
  DBSlice result;
  result.data = const_cast<char*>(s.data());
  result.len = s.size();
  return result;
}

DBSlice ToDBSlice(const DBString& s) {
  DBSlice result;
  result.data = s.data;
  result.len = s.len;
  return result;
}

DBString ToDBString(const rocksdb::Slice& s) {
  DBString result;
  result.len = s.size();
  result.data = static_cast<char*>(malloc(result.len));
  memcpy(result.data, s.data(), s.size());
  return result;
}

DBStatus ToDBStatus(const rocksdb::Status& status) {
  if (status.ok()) {
    return kSuccess;
  }
  return ToDBString(status.ToString());
}

rocksdb::ReadOptions MakeReadOptions(DBSnapshot* snap) {
  rocksdb::ReadOptions options;
  if (snap != NULL) {
    options.snapshot = snap->rep;
  }
  return options;
}

DBIterState DBIterGetState(DBIterator* iter) {
  DBIterState state;
  state.valid = iter->rep->Valid();
  if (state.valid) {
    state.key = ToDBSlice(iter->rep->key());
    state.value = ToDBSlice(iter->rep->value());
  } else {
    state.key.data = NULL;
    state.key.len = 0;
    state.value.data = NULL;
    state.value.len = 0;
  }
  return state;
}

bool WillOverflow(int64_t a, int64_t b) {
  // Morally MinInt64 < a+b < MaxInt64, but without overflows.
  // First make sure that a <= b. If not, swap them.
  if (a > b) {
    std::swap(a, b);
  }
  // Now b is the larger of the numbers, and we compare sizes
  // in a way that can never over- or underflow.
  if (b > 0) {
    return a > (std::numeric_limits<int64_t>::max() - b);
  }
  return (std::numeric_limits<int64_t>::min() - b) > a;
}

class DBLogger : public rocksdb::Logger {
 public:
  DBLogger(bool enabled)
      : enabled_(enabled) {
  }
  virtual void Logv(const char* format, va_list ap) {
    // TODO(pmattis): Benchmark calling Go exported methods from C++
    // to determine if this is too slow.
    if (!enabled_) {
      return;
    }

    // First try with a small fixed size buffer.
    char space[1024];

    // It's possible for methods that use a va_list to invalidate the data in
    // it upon use. The fix is to make a copy of the structure before using it
    // and use that copy instead.
    va_list backup_ap;
    va_copy(backup_ap, ap);
    int result = vsnprintf(space, sizeof(space), format, backup_ap);
    va_end(backup_ap);

    if ((result >= 0) && (result < sizeof(space))) {
      rocksDBLog(space, result);
      return;
    }

    // Repeatedly increase buffer size until it fits.
    int length = sizeof(space);
    while (true) {
      if (result < 0) {
        // Older behavior: just try doubling the buffer size.
        length *= 2;
      } else {
        // We need exactly "result+1" characters.
        length = result+1;
      }
      char* buf = new char[length];

      // Restore the va_list before we use it again
      va_copy(backup_ap, ap);
      result = vsnprintf(buf, length, format, backup_ap);
      va_end(backup_ap);

      if ((result >= 0) && (result < length)) {
        // It fit
        rocksDBLog(buf, result);
        delete[] buf;
        return;
      }
      delete[] buf;
    }
  }

 private:
  const bool enabled_;
};

// Getter defines an interface for retrieving a value from either an
// iterator or an engine. It is used by ProcessDeltaKey to abstract
// whether the "base" layer is an iterator or an engine.
struct Getter {
  virtual DBStatus Get(DBString* value) = 0;
};

// IteratorGetter is an implementation of the Getter interface which
// retrieves the value currently pointed to by the supplied
// iterator. It is ok for the supplied iterator to be NULL in which
// case no value will be retrieved.
struct IteratorGetter : public Getter {
  IteratorGetter(rocksdb::Iterator* iter)
      : base_(iter) {
  }

  virtual DBStatus Get(DBString* value) {
    if (base_ == NULL) {
      value->data = NULL;
      value->len = 0;
    } else {
      *value = ToDBString(base_->value());
    }
    return kSuccess;
  }

  rocksdb::Iterator* const base_;
};

// EngineGetter is an implementation of the Getter interface which
// retrieves the value for the supplied key from a DBEngine.
struct EngineGetter : public Getter {
  EngineGetter(DBEngine* db, DBSlice key)
      : db_(db),
        key_(key) {
  }

  virtual DBStatus Get(DBString* value) {
    return DBGet(db_, NULL, key_, value);
  }

  DBEngine* const db_;
  DBSlice const key_;
};

// ProcessDeltaKey performs the heavy lifting of processing the deltas
// for "key" contained in a batch and determining what the resulting
// value is. "delta" should have been seeked to "key", but may not be
// pointing to "key" if no updates existing for that key in the batch.
//
// Note that RocksDB WriteBatches append updates
// internally. WBWIIterator maintains an index for these updates on
// <key, seq-num>. Looping over the entries in WBWIIterator will
// return the keys in sorted order and, for each key, the updates as
// they were added to the batch.
DBStatus ProcessDeltaKey(Getter* base, rocksdb::WBWIIterator* delta,
                         rocksdb::Slice key, DBString* value) {
  if (value->data != NULL) {
    free(value->data);
  }
  value->data = NULL;
  value->len = 0;

  int count = 0;
  for (; delta->Valid() && delta->Entry().key == key;
       ++count, delta->Next()) {
    rocksdb::WriteEntry entry = delta->Entry();
    switch (entry.type) {
      case rocksdb::kPutRecord:
        if (value->data != NULL) {
          free(value->data);
        }
        *value = ToDBString(entry.value);
        break;
      case rocksdb::kMergeRecord: {
        DBString existing;
        if (count == 0) {
          // If this is the first record for the key, then we need to
          // merge with the record in base.
          DBStatus status = base->Get(&existing);
          if (status.data != NULL) {
            if (value->data != NULL) {
              free(value->data);
              value->data = NULL;
              value->len = 0;
            }
            return status;
          }
        } else {
          // Merge with the value we've built up so far.
          existing = *value;
          value->data = NULL;
          value->len = 0;
        }
        if (existing.data != NULL) {
          DBStatus status = DBMergeOne(
              ToDBSlice(existing), ToDBSlice(entry.value), value);
          free(existing.data);
          if (status.data != NULL) {
            return status;
          }
        } else {
          *value = ToDBString(entry.value);
        }
        break;
      }
      case rocksdb::kDeleteRecord:
        if (value->data != NULL) {
          free(value->data);
        }
        // This mirrors the logic in DBGet(): a deleted entry is
        // indicated by a value with NULL data.
        value->data = NULL;
        value->len = 0;
        break;
      default:
        break;
    }
  }

  if (count > 0) {
    return kSuccess;
  }
  return base->Get(value);
}

// This was cribbed from RocksDB and modified to support merge
// records.
class BaseDeltaIterator : public rocksdb::Iterator {
 public:
  BaseDeltaIterator(rocksdb::Iterator* base_iterator, rocksdb::WBWIIterator* delta_iterator)
      : current_at_base_(true),
        equal_keys_(false),
        status_(rocksdb::Status::OK()),
        base_iterator_(base_iterator),
        delta_iterator_(delta_iterator),
        comparator_(rocksdb::BytewiseComparator()) {
    merged_.data = NULL;
  }

  virtual ~BaseDeltaIterator() {
    ClearMerged();
  }

  bool Valid() const override {
    return current_at_base_ ? BaseValid() : DeltaValid();
  }

  void SeekToFirst() override {
    base_iterator_->SeekToFirst();
    delta_iterator_->SeekToFirst();
    UpdateCurrent();
  }

  void SeekToLast() override {
    base_iterator_->SeekToLast();
    delta_iterator_->SeekToLast();
    UpdateCurrent();
  }

  void Seek(const rocksdb::Slice& k) override {
    base_iterator_->Seek(k);
    delta_iterator_->Seek(k);
    UpdateCurrent();
  }

  void Next() override {
    if (!Valid()) {
      status_ = rocksdb::Status::NotSupported("Next() on invalid iterator");
    }
    Advance();
  }

  void Prev() override {
    status_ = rocksdb::Status::NotSupported("Prev() not supported");
  }

  rocksdb::Slice key() const override {
    return current_at_base_ ? base_iterator_->key()
                            : delta_iterator_->Entry().key;
  }

  rocksdb::Slice value() const override {
    if (current_at_base_) {
      return base_iterator_->value();
    }
    return ToSlice(merged_);
  }

  rocksdb::Status status() const override {
    if (!status_.ok()) {
      return status_;
    }
    if (!base_iterator_->status().ok()) {
      return base_iterator_->status();
    }
    return delta_iterator_->status();
  }

 private:
  // -1 -- delta less advanced than base
  // 0 -- delta == base
  // 1 -- delta more advanced than base
  int Compare() const {
    assert(delta_iterator_->Valid() && base_iterator_->Valid());
    return comparator_->Compare(delta_iterator_->Entry().key,
                                base_iterator_->key());
  }
  void AssertInvariants() {
#ifndef NDEBUG
    if (!Valid()) {
      return;
    }
    if (!BaseValid()) {
      assert(!current_at_base_ && delta_iterator_->Valid());
      return;
    }
    if (!DeltaValid()) {
      assert(current_at_base_ && base_iterator_->Valid());
      return;
    }
    // we don't support those yet
    assert(delta_iterator_->Entry().type != rocksdb::kLogDataRecord);
    int compare = comparator_->Compare(delta_iterator_->Entry().key,
                                       base_iterator_->key());
    // current_at_base -> compare < 0
    assert(!current_at_base_ || compare < 0);
    // !current_at_base -> compare <= 0
    assert(current_at_base_ && compare >= 0);
    // equal_keys_ <=> compare == 0
    assert((equal_keys_ || compare != 0) && (!equal_keys_ || compare == 0));
#endif
  }

  void Advance() {
    if (equal_keys_) {
      assert(BaseValid() && DeltaValid());
      AdvanceBase();
      AdvanceDelta();
    } else {
      if (current_at_base_) {
        assert(BaseValid());
        AdvanceBase();
      } else {
        assert(DeltaValid());
        AdvanceDelta();
      }
    }
    UpdateCurrent();
  }

  void AdvanceDelta() {
    delta_iterator_->Next();
    ClearMerged();
  }
  bool ProcessDelta() {
    IteratorGetter base(equal_keys_ ? base_iterator_.get() : NULL);
    DBStatus status = ProcessDeltaKey(&base, delta_iterator_.get(),
                                      delta_iterator_->Entry().key.ToString(),
                                      &merged_);
    if (status.data != NULL) {
      status_ = rocksdb::Status::Corruption("unable to merge records");
      free(status.data);
      return false;
    }

    // We advanced past the last entry for key and want to back up the
    // delta iterator, but we can only back up if the iterator is
    // valid.
    if (delta_iterator_->Valid()) {
      delta_iterator_->Prev();
    } else {
      delta_iterator_->SeekToLast();
    }

    return merged_.data == NULL;
  }
  void AdvanceBase() {
    base_iterator_->Next();
  }
  bool BaseValid() const { return base_iterator_->Valid(); }
  bool DeltaValid() const { return delta_iterator_->Valid(); }
  void UpdateCurrent() {
    ClearMerged();

    for (;;) {
      equal_keys_ = false;
      if (!BaseValid()) {
        // Base has finished.
        if (!DeltaValid()) {
          // Finished
          return;
        }
        if (!ProcessDelta()) {
          current_at_base_ = false;
          return;
        }
        AdvanceDelta();
        continue;
      }

      if (!DeltaValid()) {
        // Delta has finished.
        current_at_base_ = true;
        return;
      }

      int compare = Compare();
      if (compare > 0) {   // delta less than base
        current_at_base_ = true;
        return;
      }
      if (compare == 0) {
        equal_keys_ = true;
      }
      if (!ProcessDelta()) {
        current_at_base_ = false;
        return;
      }

      // Delta is less advanced and is delete.
      AdvanceDelta();
      if (equal_keys_) {
        AdvanceBase();
      }
    }

    AssertInvariants();
  }

  void ClearMerged() const {
    if (merged_.data != NULL) {
      free(merged_.data);
      merged_.data = NULL;
      merged_.len = 0;
    }
  }

  bool current_at_base_;
  bool equal_keys_;
  mutable rocksdb::Status status_;
  mutable DBString merged_;
  std::unique_ptr<rocksdb::Iterator> base_iterator_;
  std::unique_ptr<rocksdb::WBWIIterator> delta_iterator_;
  const rocksdb::Comparator* comparator_;  // not owned
};

}  // namespace

DBStatus DBOpen(DBEngine **db, DBSlice dir, DBOptions db_opts) {
  rocksdb::BlockBasedTableOptions table_options;
  table_options.block_cache = rocksdb::NewLRUCache(
      db_opts.cache_size, 4 /* num-shard-bits */);

  rocksdb::Options options;
  options.allow_os_buffer = db_opts.allow_os_buffer;
  options.compression = rocksdb::kSnappyCompression;
  //options.compaction_filter_factory.reset(new DBCompactionFilterFactory());
  options.create_if_missing = true;
  options.info_log.reset(new DBLogger(db_opts.logging_enabled));
  //options.merge_operator.reset(new DBMergeOperator);
  options.table_factory.reset(rocksdb::NewBlockBasedTableFactory(table_options));
  options.write_buffer_size = 64 << 20;           // 64 MB
  options.target_file_size_base = 64 << 20;       // 64 MB
  options.max_bytes_for_level_base = 512 << 20;   // 512 MB

  rocksdb::Env* memenv = NULL;
  if (dir.len == 0) {
    memenv = rocksdb::NewMemEnv(rocksdb::Env::Default());
    options.env = memenv;
  }

  rocksdb::DB *db_ptr;
  rocksdb::Status status = rocksdb::DB::Open(options, ToString(dir), &db_ptr);
  if (!status.ok()) {
    return ToDBStatus(status);
  }
  *db = new DBEngine;
  (*db)->rep = db_ptr;
  (*db)->memenv = memenv;
  return kSuccess;
}

DBStatus DBDestroy(DBSlice dir) {
  rocksdb::Options options;
  return ToDBStatus(rocksdb::DestroyDB(ToString(dir), options));
}

void DBClose(DBEngine* db) {
  delete db->rep;
  delete db->memenv;
  delete db;
}

DBStatus DBFlush(DBEngine* db) {
  rocksdb::FlushOptions options;
  options.wait = true;
  return ToDBStatus(db->rep->Flush(options));
}

DBStatus DBCompactRange(DBEngine* db, DBSlice* start, DBSlice* end) {
  rocksdb::Slice s;
  rocksdb::Slice e;
  rocksdb::Slice* sPtr = NULL;
  rocksdb::Slice* ePtr = NULL;
  if (start != NULL) {
    sPtr = &s;
    s = ToSlice(*start);
  }
  if (end != NULL) {
    ePtr = &e;
    e = ToSlice(*end);
  }
  return ToDBStatus(db->rep->CompactRange(rocksdb::CompactRangeOptions(), sPtr, ePtr));
}

uint64_t DBApproximateSize(DBEngine* db, DBSlice start, DBSlice end) {
  const rocksdb::Range r(ToSlice(start), ToSlice(end));
  uint64_t result;
  db->rep->GetApproximateSizes(&r, 1, &result);
  return result;
}

DBStatus DBPut(DBEngine* db, DBSlice key, DBSlice value) {
  rocksdb::WriteOptions options;
  return ToDBStatus(db->rep->Put(options, ToSlice(key), ToSlice(value)));
}

DBStatus DBMerge(DBEngine* db, DBSlice key, DBSlice value) {
  rocksdb::WriteOptions options;
  return ToDBStatus(db->rep->Merge(options, ToSlice(key), ToSlice(value)));
}

DBStatus DBGet(DBEngine* db, DBSnapshot* snap, DBSlice key, DBString* value) {
  std::string tmp;
  rocksdb::Status s = db->rep->Get(MakeReadOptions(snap), ToSlice(key), &tmp);
  if (!s.ok()) {
    if (s.IsNotFound()) {
      // This mirrors the logic in rocksdb_get(). It doesn't seem like
      // a good idea, but some code in engine_test.go depends on it.
      value->data = NULL;
      value->len = 0;
      return kSuccess;
    }
    return ToDBStatus(s);
  }
  *value = ToDBString(tmp);
  return kSuccess;
}

DBStatus DBDelete(DBEngine* db, DBSlice key) {
  rocksdb::WriteOptions options;
  return ToDBStatus(db->rep->Delete(options, ToSlice(key)));
}

DBStatus DBWrite(DBEngine* db, DBBatch *batch) {
  if (batch->updates == 0) {
    return kSuccess;
  }
  rocksdb::WriteOptions options;
  return ToDBStatus(db->rep->Write(options, batch->rep.GetWriteBatch()));
}

DBSnapshot* DBNewSnapshot(DBEngine* db)  {
  DBSnapshot *snap = new DBSnapshot;
  snap->db = db->rep;
  snap->rep = db->rep->GetSnapshot();
  return snap;
}

void DBSnapshotRelease(DBSnapshot* snap) {
  snap->db->ReleaseSnapshot(snap->rep);
  delete snap;
}

DBIterator* DBNewIter(DBEngine* db, DBSnapshot* snap) {
  DBIterator* iter = new DBIterator;
  iter->rep = db->rep->NewIterator(MakeReadOptions(snap));
  return iter;
}

void DBIterDestroy(DBIterator* iter) {
  delete iter->rep;
  delete iter;
}

DBIterState DBIterSeek(DBIterator* iter, DBSlice key) {
  iter->rep->Seek(ToSlice(key));
  return DBIterGetState(iter);
}

DBIterState DBIterSeekToFirst(DBIterator* iter) {
  iter->rep->SeekToFirst();
  return DBIterGetState(iter);
}

DBIterState DBIterSeekToLast(DBIterator* iter) {
  iter->rep->SeekToLast();
  return DBIterGetState(iter);
}

DBIterState DBIterNext(DBIterator* iter) {
  iter->rep->Next();
  return DBIterGetState(iter);
}

DBIterState DBIterPrev(DBIterator* iter){
  iter->rep->Prev();
  return DBIterGetState(iter);
}

DBStatus DBIterError(DBIterator* iter) {
  return ToDBStatus(iter->rep->status());
}

DBBatch* DBNewBatch() {
  return new DBBatch;
}

void DBBatchDestroy(DBBatch* batch) {
  delete batch;
}

void DBBatchPut(DBBatch* batch, DBSlice key, DBSlice value) {
  ++batch->updates;
  batch->rep.Put(ToSlice(key), ToSlice(value));
}

void DBBatchMerge(DBBatch* batch, DBSlice key, DBSlice value) {
  ++batch->updates;
  batch->rep.Merge(ToSlice(key), ToSlice(value));
}

DBStatus DBBatchGet(DBEngine* db, DBBatch* batch, DBSlice key, DBString* value) {
  if (batch->updates == 0) {
    return DBGet(db, NULL, key, value);
  }

  std::unique_ptr<rocksdb::WBWIIterator> iter(batch->rep.NewIterator());
  rocksdb::Slice rkey(ToSlice(key));
  iter->Seek(rkey);
  EngineGetter base(db, key);
  return ProcessDeltaKey(&base, iter.get(), rkey, value);
}

void DBBatchDelete(DBBatch* batch, DBSlice key) {
  ++batch->updates;
  batch->rep.Delete(ToSlice(key));
}

DBIterator* DBBatchNewIter(DBEngine* db, DBBatch* batch) {
  if (batch->updates == 0) {
    // Don't bother to create a batch iterator if the batch contains
    // no updates.
    return DBNewIter(db, NULL);
  }

  DBIterator* iter = new DBIterator;
  rocksdb::Iterator* base = db->rep->NewIterator(MakeReadOptions(NULL));
  rocksdb::WBWIIterator *delta = batch->rep.NewIterator();
  iter->rep = new BaseDeltaIterator(base, delta);
  return iter;
}
