package meta

import (
	"github.com/gogo/protobuf/proto"
)

// Request is an interface for RPC requests.
type Request interface {
	proto.Message
	Header() *RequestHeader
}

// Header is an common function for RequestHeader
func (rh *RequestHeader) Header() *RequestHeader {
	return rh
}

// Response is an interface for RPC responses.
type Response interface {
	proto.Message
	Header() *ResponseHeader
}

// Header is an common funciton for ResponseHeader
func (rh *ResponseHeader) Header() *ResponseHeader {
	return rh
}

// GetInner returns the Request contained in the union.
func (ru RequestUnion) GetInner() Request {
	inner := ru.GetValue()
	if inner == nil {
		return nil
	}
	return inner.(Request)
}

// GetInner returns the Response contained in the union.
func (ru ResponseUnion) GetInner() Response {
	inner := ru.GetValue()
	if inner == nil {
		return nil
	}
	return inner.(Response)
}
