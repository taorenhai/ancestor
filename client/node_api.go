package client

import (
	"reflect"

	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/meta"
)

// nodeAPI is a database handle to a single ancestordb cluster.
type nodeAPI struct {
	sender Sender
	nodeID meta.NodeID
}

// newNodeAPI return new node client
func newNodeAPI(sender Sender, nodeID meta.NodeID) *nodeAPI {
	return &nodeAPI{
		sender: sender,
		nodeID: nodeID,
	}
}

func (n *nodeAPI) getNodeID() meta.NodeID {
	return n.nodeID
}

// getAddr return the Romote server addr
func (n *nodeAPI) getAddr() string {
	return n.sender.getAddr()
}

// close the sender
func (n *nodeAPI) stop() {
	n.sender.stop()
}

// beginTransaction use to start a transaction
func (n *nodeAPI) beginTransaction(txn *meta.Transaction, id meta.RangeID) error {
	req := &meta.BeginTransactionRequest{
		RequestHeader: meta.RequestHeader{
			Key:       meta.Key(txn.ID),
			Txn:       txn,
			RangeID:   id,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsWrite,
		},
	}

	_, err := runWithResult(n.sender, req)

	return err
}

// endTransaction comimt or rollback
func (n *nodeAPI) endTransaction(txn *meta.Transaction, id meta.RangeID, commit bool) error {
	req := &meta.EndTransactionRequest{
		RequestHeader: meta.RequestHeader{
			Key:       meta.Key(txn.ID),
			Txn:       txn,
			RangeID:   id,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsWrite,
		},
		Commit: commit,
	}

	_, err := runWithResult(n.sender, req)

	return err
}

// resolveIntent clear key's meta data
func (n *nodeAPI) resolveIntent(txn *meta.Transaction, id meta.RangeID, keys []meta.Key) error {
	req := &meta.ResolveIntentRequest{
		RequestHeader: meta.RequestHeader{
			Key:       keys[0],
			Txn:       txn,
			RangeID:   id,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsWrite,
		},
	}

	for _, key := range keys {
		it := meta.Intent{
			Span: meta.Span{Key: key},
			Txn:  *txn,
		}

		req.Intents = append(req.Intents, it)
	}

	_, err := runWithResult(n.sender, req)

	return err
}

// scan retrieves the rows between begin (inclusive) and end (exclusive) in
// ascending order.
func (n *nodeAPI) scan(txn *meta.Transaction, rangeID meta.RangeID, begin, end meta.Key, maxRows int64) ([]meta.KeyValue, error) {
	req := &meta.ScanRequest{
		RequestHeader: meta.RequestHeader{
			Key:       begin,
			EndKey:    end,
			Txn:       txn,
			RangeID:   rangeID,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsRead,
		},
		MaxResults: maxRows,
	}

	resp, err := runWithResult(n.sender, req)
	if err != nil {
		return []meta.KeyValue{}, err
	}

	scanResp, ok := resp.(*meta.ScanResponse)
	if !ok {
		return nil, errors.Trace(meta.NewInvalidResponseTypeError("*meta.ScanRequest", reflect.TypeOf(req).String()))
	}

	rows := make([]meta.KeyValue, len(scanResp.Rows))
	for i := range scanResp.Rows {
		rows[i].Key = scanResp.Rows[i].Key
		rows[i].Value = scanResp.Rows[i].Value
	}

	return rows, nil
}

// Scan retrieves the rows between begin (inclusive) and end (exclusive) in
// descending order.
func (n *nodeAPI) reverseScan(txn *meta.Transaction, rangeID meta.RangeID, begin, end meta.Key, maxRows int64) ([]meta.KeyValue, error) {
	req := &meta.ReverseScanRequest{
		RequestHeader: meta.RequestHeader{
			Key:       begin,
			EndKey:    end,
			Txn:       txn,
			RangeID:   rangeID,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsRead,
		},
		MaxResults: maxRows,
	}

	resp, err := runWithResult(n.sender, req)
	if err != nil {
		return []meta.KeyValue{}, err
	}

	reverseScanResp, ok := resp.(*meta.ReverseScanResponse)
	if !ok {
		return nil, errors.Trace(meta.NewInvalidResponseTypeError("*meta.ReverseScanRequest", reflect.TypeOf(req).String()))
	}

	rows := make([]meta.KeyValue, len(reverseScanResp.Rows))
	for i := range reverseScanResp.Rows {
		rows[i].Key = reverseScanResp.Rows[i].Key
		rows[i].Value = reverseScanResp.Rows[i].Value
	}

	return rows, nil
}

//PushTxn send timestamp,key and txn
func (n *nodeAPI) pushTxn(pushType meta.PushTxnType, id meta.RangeID, key meta.Key, txn *meta.Transaction, latestTimestamp meta.Timestamp) (*meta.Transaction, error) {
	req := &meta.PushTransactionRequest{
		RequestHeader: meta.RequestHeader{
			Txn:       txn,
			RangeID:   id,
			Timestamp: txn.GetTimestamp(),
			Flag:      meta.IsWrite,
			Key:       key,
		},
		PusheeKey:       key,
		PushType:        pushType,
		LatestTimestamp: latestTimestamp,
	}

	resp, err := runWithResult(n.sender, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	pushResp, ok := resp.(*meta.PushTransactionResponse)
	if !ok {
		return nil, errors.Trace(meta.NewInvalidResponseTypeError("meta.PushTransactionResponse",
			reflect.TypeOf(req).String()))
	}

	return pushResp.GetPusheeTxn(), nil
}

//add raft node configure
func (n *nodeAPI) addRaftConf(start, end meta.Key, nodeID meta.NodeID, id meta.RangeID) error {
	req := &meta.AddRaftConfRequest{
		RequestHeader: meta.RequestHeader{
			Key:     start,
			EndKey:  end,
			RangeID: id,
			Flag:    meta.IsLeader,
		},
		NodeID: nodeID,
	}

	_, err := runWithResult(n.sender, req)
	return err
}

func (n *nodeAPI) compactLog(start, end meta.Key, id meta.RangeID) error {
	req := &meta.CompactLogRequest{
		RequestHeader: meta.RequestHeader{
			Key:     start,
			EndKey:  end,
			RangeID: id,
			Flag:    meta.IsLeader,
		},
	}
	_, err := runWithResult(n.sender, req)
	return err
}

func (n *nodeAPI) createReplica(position, rd meta.RangeDescriptor) error {
	req := &meta.CreateReplicaRequest{
		RequestHeader: meta.RequestHeader{
			Key:     position.StartKey,
			EndKey:  position.EndKey,
			RangeID: position.RangeID,
			Flag:    meta.IsSystem,
		},
		RangeDescriptor: rd,
		NodeID:          n.nodeID,
	}

	_, err := runWithResult(n.sender, req)
	return err
}
