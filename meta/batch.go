package meta

import (
	"fmt"

	"github.com/juju/errors"
)

// Executer BatchRequest interface
type Executer interface {
	// ExecuteCmd request wait response, or return error
	ExecuteCmd(*BatchRequest) (*BatchResponse, error)
}

// IsLeader returns true if the BatchRequest contains an leader request.
func (br *BatchRequest) IsLeader() bool {
	return br.Flags()&IsLeader != 0
}

// IsWrite returns true if the BatchRequest contains a write.
func (br *BatchRequest) IsWrite() bool {
	return br.Flags()&IsWrite != 0
}

// IsReadOnly returns true if all requests within are read-only.
func (br *BatchRequest) IsReadOnly() bool {
	flags := br.Flags()
	return (flags&IsRead != 0) && (flags&IsWrite == 0)
}

// IsSystem returns true if all requests within are system.
func (br *BatchRequest) IsSystem() bool {
	return br.Flags()&IsSystem != 0
}

// First returns the first response of the given type, if possible.
func (br *BatchResponse) First() Response {
	if len(br.Resp) > 0 {
		return br.Resp[0].GetInner()
	}
	return nil
}

// Add adds a request to the batch request.
func (br *BatchRequest) Add(requests ...Request) {
	for _, args := range requests {
		union := RequestUnion{}
		if !union.SetValue(args) {
			panic(fmt.Sprintf("unable to add %T to batch request", args))
		}
		br.Req = append(br.Req, union)
	}
}

// AddUnion add a RequestUnion to BatchRequest
func (br *BatchRequest) AddUnion(union RequestUnion) {
	br.Req = append(br.Req, union)
}

// Add adds a response to the batch response.
func (br *BatchResponse) Add(reply Response) {
	union := ResponseUnion{}
	if !union.SetValue(reply) {
		// TODO(tschottdorf) evaluate whether this should return an error.
		panic(fmt.Sprintf("unable to add %T to batch response", reply))
	}
	br.Resp = append(br.Resp, union)
}

// Flags BatchRequest flags
func (br *BatchRequest) Flags() RequestFlag {
	var flags RequestFlag
	for _, union := range br.Req {
		flags |= union.GetInner().Header().Flag
	}
	return flags
}

// GetPushType return the PushTxnType
func (br *BatchRequest) GetPushType() PushTxnType {
	if br.IsWrite() {
		return PUSH_ABORT
	}

	return PUSH_TIMESTAMP
}

// SetNewRequest set a new Request
func (br *BatchRequest) SetNewRequest() {
	if br.Txn == nil {
		return
	}
	br.Txn.Sequence++
}

// Execute execute internal request in sequence
func (br *BatchRequest) Execute(execute func(Request) (Response, error)) (*BatchResponse, error) {
	batchResp := &BatchResponse{}
	for _, unionReq := range br.GetReq() {
		req := unionReq.GetInner()
		if req == nil {
			return batchResp, NewError(errors.Errorf("bad Request, need inner Request"))
		}
		resp, err := execute(req)
		if err != nil {
			return batchResp, errors.Trace(err)
		}
		batchResp.Add(resp)
	}
	return batchResp, nil
}
