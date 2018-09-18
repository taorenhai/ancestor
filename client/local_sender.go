package client

import (
	"fmt"

	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/meta"
)

type localSender struct {
	nodeID   meta.NodeID
	executer meta.Executer
}

func newLocalSender(id meta.NodeID, executer meta.Executer) Sender {
	return &localSender{
		nodeID:   id,
		executer: executer,
	}
}

func (l *localSender) send(req *meta.BatchRequest, resp *meta.BatchResponse) error {
	bresp, err := l.executer.ExecuteCmd(req)
	if err != nil {
		return errors.Trace(err)
	}

	*resp = *bresp
	return nil
}

func (l *localSender) stop() {
}

func (l *localSender) getAddr() string {
	return fmt.Sprintf("local nodeID:%d", l.nodeID)
}
