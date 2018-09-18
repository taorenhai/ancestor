package client

import (
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

type rangeKeyMap struct {
	rangeID meta.RangeID
	keys    map[string]meta.Key
}

//resolveIntent send resolveintent
type resolveIntent struct {
	pool map[meta.RangeID]*rangeKeyMap
	*base
	*factory
}

func newResolveIntent(b *base, f *factory) *resolveIntent {
	return &resolveIntent{
		pool:    make(map[meta.RangeID]*rangeKeyMap),
		base:    b,
		factory: f,
	}
}

func (ri *resolveIntent) add(rangeID meta.RangeID, key meta.Key) {
	rkm, ok := ri.pool[rangeID]
	if !ok {
		rkm = &rangeKeyMap{
			rangeID: rangeID,
			keys:    make(map[string]meta.Key),
		}
		ri.pool[rangeID] = rkm
	}

	rkm.keys[key.String()] = meta.Key(append([]byte{}, key...))
}

func (rkm *rangeKeyMap) dump() []meta.Key {
	keyList := []meta.Key{}
	for _, val := range rkm.keys {
		keyList = append(keyList, val)
	}
	return keyList
}

//send resolveIntent and clear key cache
func (ri *resolveIntent) send(txn *dbTxn) {
	for _, keys := range ri.pool {
		keyList := keys.dump()
		if len(keyList) > 0 {
			ri.asyncRun(func(w *worker) {
				log.Debugf("worker:%s will ResolveIntent keySize:%d", w.name, len(keyList))
				ri.doNodeRequest(keyList[0], func(node *nodeAPI, rd meta.RangeDescriptor) error {
					err := node.resolveIntent(&txn.Transaction, rd.RangeID, keyList)
					if err != nil {
						log.Errorf("%s send ResolveIntent to %s, error:%s", txn.Name, node.getAddr(), err.Error())
						return err
					}
					log.Debugf("%s resolveIntent keys:%s", txn.Name, keyList)
					return nil
				})
			})
		}
	}
	ri.pool = nil
}
