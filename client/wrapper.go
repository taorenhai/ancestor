package client

import (
	"reflect"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

func runWithResult(sender Sender, request meta.Request) (meta.Response, error) {
	resp := &meta.BatchResponse{}
	var req *meta.BatchRequest
	var ok bool

	if req, ok = request.(*meta.BatchRequest); !ok {
		req = &meta.BatchRequest{}
		req.Txn = request.Header().GetTxn()
		req.Timestamp = request.Header().GetTimestamp()
		req.RangeID = request.Header().RangeID
		req.Add(request)
	}

	if err := sender.send(req, resp); err != nil {
		log.Errorf("%s Send(%s) error:%s", reflect.TypeOf(request).String(), sender.getAddr(), err.Error())
		return nil, errors.Trace(err)
	}

	if err := resp.GetError(); err != nil {
		switch e := err.GoError().(type) {
		case *meta.WriteTooOldError:
			return nil, errors.Trace(meta.NewNeedRetryError(e.String()))
		case *meta.TransactionStatusError:
			if e.Retry {
				return nil, errors.Trace(meta.NewNeedRetryError(e.String()))
			}
		}
		return nil, errors.Trace(err.GoError())
	}

	if _, ok = request.(*meta.BatchRequest); !ok {
		return resp.GetResp()[0].GetInner().(meta.Response), nil
	}
	return resp, nil
}
