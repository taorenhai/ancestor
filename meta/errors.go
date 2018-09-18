package meta

import (
	"fmt"
)

// ResponseWithError is a tuple of a BatchResponse and an error. It is used to
// pass around a BatchResponse with its associated error where that
// entanglement is necessary (e.g. channels, methods that need to return
// another error in addition to this one).
type ResponseWithError struct {
	Reply *BatchResponse
	Err   error
}

func (e Error) getDetail() error {
	if e.Detail == nil {
		return nil
	}
	return e.Detail.GetValue().(error)
}

// NewError creates an Error from the given error.
func NewError(err error) *Error {
	if err == nil {
		return nil
	}
	e := &Error{}
	if intErr, ok := err.(*Error); ok {
		*e = *(*Error)(intErr)
	} else {
		e.SetGoError(err)
	}
	return e
}

// String implements fmt.Stringer.
func (e *Error) String() string {
	return e.Message
}

// Error implements error.
func (e *Error) Error() string {
	return (*Error)(e).String()
}

// GoError returns the non-nil error from the meta.Error union.
func (e *Error) GoError() error {
	if e == nil {
		return nil
	}

	if e.Detail == nil {
		return (*Error)(e)
	}
	err := e.getDetail()
	if err == nil {
		// Unknown error detail; return the generic error.
		return (*Error)(e)
	}

	return err
}

// SetGoError sets Error using err.
func (e *Error) SetGoError(err error) {
	if e.Message != "" {
		panic("cannot re-use Error")
	}
	e.Message = err.Error()
	// If the specific error type exists in the detail union, set it.
	detail := &ErrorDetail{}
	if detail.SetValue(err) {
		e.Detail = detail
	}
}

// NewWriteIntentError set WriteIntentError
func NewWriteIntentError(pusherTxn *Transaction, pushType PushTxnType, pusheeTxnID, pusheekey Key) *WriteIntentError {
	return &WriteIntentError{
		Intent:    Intent{Span: Span{Key: pusheekey}, Txn: Transaction{ID: pusheeTxnID}},
		PusherTxn: pusherTxn,
		PushType:  pushType,
	}
}

// Error formats error.
func (e *WriteIntentError) Error() string {
	return fmt.Sprintf("conflicting intents on %v", e.Intent.Key)
}

// NewWriteTooOldError initializes a new WriteTooOldError
func NewWriteTooOldError(timestamp Timestamp, existingTimestamp Timestamp) *WriteTooOldError {
	return &WriteTooOldError{Timestamp: timestamp, ExistingTimestamp: existingTimestamp}
}

// Error formats error.
func (e *WriteTooOldError) Error() string {
	return fmt.Sprintf("write too old: timestamp %s <= %s", e.Timestamp.String(), e.ExistingTimestamp.String())
}

// NewInvalidResponseTypeError initializes a new InvalidResponseTypeError.
func NewInvalidResponseTypeError(wantType string, actualType string) *InvalidResponseTypeError {
	return &InvalidResponseTypeError{WantType: wantType, ActualType: actualType}
}

// Error formats error
func (e *InvalidResponseTypeError) Error() string {
	return fmt.Sprintf("response type is invalid, wantType:%s actualType:%s", e.WantType, e.ActualType)
}

// NewNotLeaderError initializes a new NotLeaderError.
func NewNotLeaderError(nodeID NodeID) *NotLeaderError {
	return &NotLeaderError{NodeID: nodeID}
}

// Error formats error
func (e *NotLeaderError) Error() string {
	return fmt.Sprintf("current server not leader, leader nodeID:%v", e.NodeID)
}

// NewInvalidKeyError initializes a new InvalidKeyError.
func NewInvalidKeyError() *InvalidKeyError {
	return &InvalidKeyError{}
}

// Error formats error
func (e *InvalidKeyError) Error() string {
	return fmt.Sprintf("key is invalid")
}

// NewRangeNotFoundError initializes a new RangeNotFoundError
func NewRangeNotFoundError(rangeID RangeID) *RangeNotFoundError {
	return &RangeNotFoundError{RangeID: rangeID}
}

// Error formats error
func (e *RangeNotFoundError) Error() string {
	return fmt.Sprintf("range:%d not found", e.RangeID)
}

// NewTransactionStatusError initializes a new TransactionStatusError
func NewTransactionStatusError(txn *Transaction, msg string, retry bool) *TransactionStatusError {
	return &TransactionStatusError{Txn: *txn.Clone(), Msg: msg, Retry: retry}
}

// Error formats error
func (e *TransactionStatusError) Error() string {
	return fmt.Sprintf("Transaction status error, txn:%v msg:%v retry:%v", e.Txn.String(), e.Msg, e.Retry)
}

// NewUnmarshalDataError initializes a new UnmarshalDataError
func NewUnmarshalDataError(data []byte, msg string) *UnmarshalDataError {
	return &UnmarshalDataError{Data: data, Msg: msg}
}

// Error formats error
func (e *UnmarshalDataError) Error() string {
	return fmt.Sprintf("Unmarshal data error, data:%q msg:%s", e.Data, e.Msg)
}

// NewMarshalDataError initializes a new MarshalDataError
func NewMarshalDataError(msg string) *MarshalDataError {
	return &MarshalDataError{Msg: msg}
}

// Error formats error
func (e *MarshalDataError) Error() string {
	return fmt.Sprintf("Marshal data error, msg:%s", e.Msg)
}

// NewEnginePutDataError initializes a new EnginePutDataError
func NewEnginePutDataError(key []byte, data []byte, msg string) *EnginePutDataError {
	return &EnginePutDataError{Key: key, Data: data, Msg: msg}
}

// Error formats error
func (e *EnginePutDataError) Error() string {
	return fmt.Sprintf("engine put data error, key:%q data:%q msg:%s", e.Key, e.Data, e.Msg)
}

// NewEngineGetDataError initializes a new EngineGetDataError
func NewEngineGetDataError(key []byte, msg string) *EngineGetDataError {
	return &EngineGetDataError{Key: key, Msg: msg}
}

// Error formats error
func (e *EngineGetDataError) Error() string {
	return fmt.Sprintf("engine get data error, key:%q msg:%s", e.Key, e.Msg)
}

// NewEngineClearDataError initializes a new EngineClearDataError
func NewEngineClearDataError(key []byte, msg string) *EngineClearDataError {
	return &EngineClearDataError{Key: key, Msg: msg}
}

// Error formats error
func (e *EngineClearDataError) Error() string {
	return fmt.Sprintf("engine clear data error, key:%q msg:%s", e.Key, e.Msg)
}

// NewInvalidTransactionError initializes a new InvalidTransactionError
func NewInvalidTransactionError(msg string) *InvalidTransactionError {
	return &InvalidTransactionError{Msg: msg}
}

// Error formats error
func (e *InvalidTransactionError) Error() string {
	return fmt.Sprintf("invalid transaction error, msg:%v", e.Msg)
}

// NewInvalidNilError initializes a new InvalidNilError
func NewInvalidNilError(msg string) *InvalidNilDataError {
	return &InvalidNilDataError{Msg: msg}
}

// Error formats error
func (e *InvalidNilDataError) Error() string {
	return fmt.Sprintf("the param is nil, msg:%v", e.Msg)
}

// Error formats error
func (e *HeartbeatTransactionError) Error() string {
	return fmt.Sprintf("Heartbeat has a Error")
}

// Error formats error
func (e *DecodeBytesError) Error() string {
	return fmt.Sprintf("there should be 8 bytes for encoded timestamp: %q key: %q", e.TsBytes, e.Key)
}

// NewTransactionAbortedError initializes a new TransactionAbortedError
func NewTransactionAbortedError(txn *Transaction) *TransactionAbortedError {
	return &TransactionAbortedError{Txn: *txn.Clone()}
}

// Error formats error
func (e *TransactionAbortedError) Error() string {
	return fmt.Sprintf("transaction aborted error, txn:%v ", e.Txn.String())
}

// RPCConnectError rpc connect failed or close
type RPCConnectError struct {
	host string
}

// NewRPCConnectError initializes a new  rpc connect error
func NewRPCConnectError(host string) *RPCConnectError {
	return &RPCConnectError{host: host}
}

// Error formats error
func (e *RPCConnectError) Error() string {
	return fmt.Sprintf("Connect to %s error", e.host)
}

// ReadPriorityError When a read request return TransactionAbortedError
type ReadPriorityError struct {
}

// NewReadPriorityError initializes a new ReadPriorityError
func NewReadPriorityError() *ReadPriorityError {
	return &ReadPriorityError{}
}

// Error formats error
func (e *ReadPriorityError) Error() string {
	return ""
}

// NewNeedRetryError initializes a new NeedRetry error.
func NewNeedRetryError(m string) *NeedRetryError {
	return &NeedRetryError{Msg: m}
}

// Error formats error
func (e *NeedRetryError) Error() string {
	return e.Msg
}

// NewNotExistError initializes a new not exist error.
func NewNotExistError(k Key) *NotExistError {
	return &NotExistError{Msg: fmt.Sprintf("key:%q not exist", k)}
}

// Error formats error
func (e *NotExistError) Error() string {
	return e.Msg
}
