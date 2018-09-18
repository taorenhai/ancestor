package client

import (
	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/meta"
)

type batchRequest struct {
	meta.BatchRequest
}

func newBatchRequest(key meta.Key, rangeID meta.RangeID) *batchRequest {
	return &batchRequest{
		BatchRequest: meta.BatchRequest{
			RequestHeader: meta.RequestHeader{
				RangeID: rangeID,
				Key:     key,
			},
		},
	}
}

// base method to create a GetRequest
func (br *batchRequest) addGet(txn *meta.Transaction, key meta.Key) {
	req := &meta.GetRequest{
		RequestHeader: meta.RequestHeader{
			Key:       key,
			Txn:       txn,
			RangeID:   br.RangeID,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsRead,
		},
	}

	br.Add(req)
}

// base method to create a PutRequest
func (br *batchRequest) addPut(txn *meta.Transaction, key meta.Key, value meta.Value) {
	req := &meta.PutRequest{
		RequestHeader: meta.RequestHeader{
			Key:       key,
			Txn:       txn,
			RangeID:   br.RangeID,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsWrite,
		},
		Value: value,
	}

	br.Add(req)
}

// create a transaction heartbeat
func (br *batchRequest) addHeartbeat(txn *meta.Transaction) {
	req := &meta.HeartbeatTransactionRequest{
		RequestHeader: meta.RequestHeader{
			Key:       meta.Key(txn.ID),
			Txn:       txn,
			RangeID:   br.RangeID,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsWrite,
		},
	}

	br.Add(req)
}

// runBatchWithResult - process meta.BatchRequest and return the result
func (br *batchRequest) runBatchWithResult(node *nodeAPI) ([]meta.Response, error) {
	resp, err := runWithResult(node.sender, &br.BatchRequest)
	if err != nil {
		return nil, errors.Trace(err)
	}

	result := []meta.Response{}
	respBatch, ok := resp.(*meta.BatchResponse)
	if !ok {
		return nil, errors.Trace(errors.New("response to batchresponse error"))
	}
	for _, subResp := range respBatch.GetResp() {
		subRespInner := subResp.GetInner()
		result = append(result, subRespInner)
	}

	return result, nil
}
