package client

import (
	"math"
	"strconv"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/retry"
)

var (
	// MaxVersion max timestamp.
	MaxVersion = meta.Timestamp{Logical: math.MaxInt32, WallTime: math.MaxInt64}
)

// IsRetryableError checks if the err is a fatal error and the under going operation is worth to retry.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.Cause(err).(*meta.NeedRetryError); ok {
		return true
	}
	return false
}

// IsErrNotFound checks if err is a kind of NotFound error.
func IsErrNotFound(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.Cause(err).(*meta.NotExistError); ok {
		return true
	}

	return false
}

// RunInNewTxn will run the f in a new transaction environment.
func RunInNewTxn(store Storage, retryable bool, f func(Transaction) error) error {
	var (
		err error
		txn Transaction
	)

	for r := retry.Start(defaultRetryOptions); r.Next(); {
		txn, err = store.Begin(r.CurrentAttempt())
		if err != nil {
			log.Errorf("[kv] RunInNewTxn error - %v", err)
			return errors.Trace(err)
		}

		err = f(txn)
		if retryable && IsRetryableError(err) {
			log.Warnf("[kv] Retry txn %v", txn)
			txn.Rollback()
			continue
		}
		if err != nil {
			txn.Rollback()
			return errors.Trace(err)
		}

		err = txn.Commit()
		if retryable && IsRetryableError(err) {
			log.Warnf("[kv] Retry txn %v", txn)
			txn.Rollback()
			continue
		}
		if err != nil {
			return errors.Trace(err)
		}
		break
	}
	return errors.Trace(err)
}

// IncInt64 increases the value for key k in kv store by step.
func IncInt64(txn Transaction, k meta.Key, step int64) (int64, error) {
	val, err := txn.Get(k)
	if IsErrNotFound(err) {
		err = txn.Set(k, []byte(strconv.FormatInt(step, 10)))
		if err != nil {
			return 0, errors.Trace(err)
		}
		return step, nil
	}

	if err != nil {
		return 0, errors.Trace(err)
	}

	intVal, err := strconv.ParseInt(string(val), 10, 0)
	if err != nil {
		return 0, errors.Trace(err)
	}

	intVal += step
	err = txn.Set(k, []byte(strconv.FormatInt(intVal, 10)))
	if err != nil {
		return 0, errors.Trace(err)
	}
	return intVal, nil
}

// GetInt64 get int64 value which created by IncInt64 method.
func GetInt64(r Retriever, k meta.Key) (int64, error) {
	val, err := r.Get(k)
	if IsErrNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Trace(err)
	}

	intVal, err := strconv.ParseInt(string(val), 10, 0)
	return intVal, errors.Trace(err)
}
