package client

import (
	"net"
	"reflect"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

// pdAPI is a database handle to a single ancestordb cluster.
type pdAPI struct {
	sender Sender
}

// newPDAPI return new pd client
func newPDAPI(sender Sender) *pdAPI {
	return &pdAPI{sender: sender}
}

func (pd *pdAPI) reconnect(addr net.Addr) {
	if pd.sender != nil {
		pd.sender.stop()
	}

	pd.sender = newRPCSender(addr, defaultRetryOptions)
}

// getAddr return the Romote server addr
func (pd *pdAPI) getAddr() string {
	return pd.sender.getAddr()
}

// close the sender
func (pd *pdAPI) stop() {
	pd.sender.stop()
}

func (pd *pdAPI) getTimestamp() (meta.Timestamp, error) {
	req := &meta.GetTimestampRequest{}
	resp, err := runWithResult(pd.sender, req)
	if err != nil {
		return meta.Timestamp{}, errors.Trace(err)
	}

	gr, ok := resp.(*meta.GetTimestampResponse)
	if !ok {
		return meta.Timestamp{}, errors.Trace(meta.NewInvalidResponseTypeError("*meta.GetTimestampRequest",
			reflect.TypeOf(req).String()))
	}

	return gr.GetTimestamp(), nil
}

// getRangeDescriptors get rangeDescriptors with keys, return all RangeDescriptors when keys is nil
func (pd *pdAPI) getRangeDescriptors(keys ...meta.Key) ([]meta.RangeDescriptor, error) {
	req := &meta.GetRangeDescriptorsRequest{}

	if len(keys) > 0 && len(keys[0]) > 0 {
		req.Key = keys[0]
	}

	resp, err := runWithResult(pd.sender, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	gr, ok := resp.(*meta.GetRangeDescriptorsResponse)
	if !ok {
		return nil, errors.Trace(meta.NewInvalidResponseTypeError("*meta.GetRangeDescriptorsRequest",
			reflect.TypeOf(req).String()))
	}

	var rds []meta.RangeDescriptor
	for _, v := range gr.GetRangeDescriptor() {
		rds = append(rds, *v)
	}
	log.Debugf("load rds:%+v", rds)

	return rds, nil
}

// setRangeDescriptors set rangeDescriptor to pd
func (pd *pdAPI) setRangeDescriptors(descs []meta.RangeDescriptor) error {
	req := &meta.SetRangeDescriptorsRequest{
		RangeDescriptor: descs,
	}

	_, err := runWithResult(pd.sender, req)
	return err
}

// setNodeStats send stats info to pd server
func (pd *pdAPI) setNodeStats(stats meta.NodeStatsInfo) error {
	req := &meta.SetNodeStatsRequest{
		NodeStatsInfo: stats,
	}

	_, err := runWithResult(pd.sender, req)
	return err
}

// setRangeStats send stats info to pd server
func (pd *pdAPI) setRangeStats(stats []meta.RangeStatsInfo) error {
	req := &meta.SetRangeStatsRequest{
		RangeStatsInfo: stats,
	}

	_, err := runWithResult(pd.sender, req)
	return err
}

// GetRangeStats query the stats info from pd
func (pd *pdAPI) getRangeStats(rangeIDs []meta.RangeID) ([]meta.RangeStatsInfo, error) {
	req := &meta.GetRangeStatsRequest{
		RangeIDs: rangeIDs,
	}

	resp, err := runWithResult(pd.sender, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	gr, ok := resp.(*meta.GetRangeStatsResponse)
	if !ok {
		return nil, errors.Trace(meta.NewInvalidResponseTypeError("*meta.GetRangeStatsRequest",
			reflect.TypeOf(req).String()))
	}

	list := []meta.RangeStatsInfo{}
	for _, row := range gr.GetRows() {
		list = append(list, *row)
	}

	return list, nil
}

func (pd *pdAPI) setNodeDescriptors(nodeDescs []meta.NodeDescriptor) error {
	req := &meta.SetNodeDescriptorsRequest{
		NodeDescriptor: nodeDescs,
	}

	_, err := runWithResult(pd.sender, req)
	return err
}

func (pd *pdAPI) getNodeDescriptors() ([]meta.NodeDescriptor, error) {
	req := &meta.GetNodeDescriptorsRequest{}

	resp, err := runWithResult(pd.sender, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	gr, ok := resp.(*meta.GetNodeDescriptorsResponse)
	if !ok {
		return nil, errors.Trace(meta.NewInvalidResponseTypeError("*meta.GetNodeDescriptorsRequest",
			reflect.TypeOf(req).String()))
	}

	var nodeDescs []meta.NodeDescriptor
	for _, v := range gr.GetNodeDescriptor() {
		nodeDescs = append(nodeDescs, *v)
	}

	return nodeDescs, err
}

func (pd *pdAPI) getNewID() (meta.RangeID, error) {
	req := &meta.GetNewIDRequest{}

	resp, err := runWithResult(pd.sender, req)
	if err != nil {
		return meta.RangeID(0), errors.Trace(err)
	}

	gr, ok := resp.(*meta.GetNewIDResponse)
	if !ok {
		return meta.RangeID(0), errors.Trace(meta.NewInvalidResponseTypeError("*meta.GetNewIDRequest",
			reflect.TypeOf(req).String()))
	}

	return gr.ID, nil
}

func (pd *pdAPI) bootstrap(addr string, capacity int64) (meta.NodeDescriptor, error) {
	req := &meta.BootstrapRequest{
		Address:  addr,
		Capacity: capacity,
	}

	resp, err := runWithResult(pd.sender, req)
	if err != nil {
		return meta.NodeDescriptor{}, errors.Trace(err)
	}

	br, ok := resp.(*meta.BootstrapResponse)
	if !ok {
		return meta.NodeDescriptor{}, errors.Trace(meta.NewInvalidResponseTypeError("*meta.BootstrapRequest",
			reflect.TypeOf(req).String()))
	}
	return br.NodeDescriptor, nil
}
