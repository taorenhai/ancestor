package client

import (
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

const (
	heartbeatInterval = time.Second
)

type heartbeat struct {
	addChn chan meta.Transaction
	delChn chan string
	txnMap map[string]meta.Transaction
	isStop bool

	sync.Mutex
	*base
	*factory
	*admin
}

func newHeartbeat(b *base, f *factory, a *admin) *heartbeat {
	return &heartbeat{
		txnMap:  make(map[string]meta.Transaction),
		addChn:  make(chan meta.Transaction, 1),
		delChn:  make(chan string, 1),
		isStop:  true,
		base:    b,
		factory: f,
		admin:   a,
	}
}

func (h *heartbeat) add(txn meta.Transaction) {
	log.Debugf("add txn id:%s", txn.ID)
	h.Lock()
	if !h.isStop {
		h.addChn <- txn
	}
	h.Unlock()
}

func (h *heartbeat) del(txnID string) {
	log.Debugf("del txn id:%s", txnID)
	h.Lock()
	if !h.isStop {
		h.delChn <- txnID
	}
	h.Unlock()
}

func (h *heartbeat) sendHeartbeat(txns []meta.Transaction, lastTimestamp meta.Timestamp) {
	brs := make(map[meta.RangeID]*batchRequest)

	for i := range txns {
		key := meta.Key(txns[i].ID)
		rd, err := h.getRangeDescFromCache(key)
		if err != nil {
			log.Errorf("getRangeDescFromCache key:%q error:%s", key, errors.ErrorStack(err))
			continue
		}

		txns[i].LastHeartbeat = lastTimestamp.WallTime

		br, ok := brs[rd.RangeID]
		if !ok {
			br = newBatchRequest(key, rd.RangeID)
			brs[rd.RangeID] = br
		}

		br.addHeartbeat(&txns[i])
	}

	for _, br := range brs {
		err := h.doNodeRequest(br.Key, func(node *nodeAPI, rd meta.RangeDescriptor) error {
			resps, err := br.runBatchWithResult(node)
			if err != nil {
				return err
			}
			for _, resp := range resps {
				hr, _ := resp.(*meta.HeartbeatTransactionResponse)
				if txn := hr.GetTxn(); txn != nil {
					log.Debugf("heartbeat remove txn:%s", txn.ID)
					h.del(string(txn.ID))
				}
			}
			return nil
		})
		if err != nil {
			log.Errorf("heartbeat error:%s", errors.ErrorStack(err))
		}
	}
}

func (h *heartbeat) start() {
	go h.run()
}

func (h *heartbeat) run() {
	t := time.NewTicker(heartbeatInterval)

	for {
		select {
		case <-t.C:
			h.Lock()
			if h.isStop {
				t.Stop()
				h.Unlock()
				return
			}
			h.Unlock()

			txns := []meta.Transaction{}
			for key := range h.txnMap {
				txns = append(txns, h.txnMap[key])
			}

			if len(txns) > 0 {
				h.asyncRun(func(w *worker) {
					if t, err := h.GetTimestamp(); err == nil {
						log.Debugf("worker:%s will sendHeartbeat, lastTimestamp:%v", w.name, t)
						h.sendHeartbeat(txns, t)
					}
				})
			}

		case txn := <-h.addChn:
			h.txnMap[string(txn.ID)] = txn

		case key := <-h.delChn:
			delete(h.txnMap, key)
		}
	}
}

func (h *heartbeat) stop() {
	h.Lock()
	h.isStop = true
	h.Unlock()
}
