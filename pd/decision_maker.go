package pd

import (
	"github.com/taorenhai/ancestor/meta"
)

type strategy interface {
	checkSplit(*meta.RangeStatsInfo, uint64, float32) bool
}

type defaultStrategy struct {
}

func (d *defaultStrategy) checkSplit(r *meta.RangeStatsInfo, cap uint64, threshold float32) bool {
	if r.IsRaftLeader && !r.IsRemoved && r.TotalBytes > int64(float32(cap)*threshold) {
		return true
	}

	return false
}

type decisionMaker struct {
	strategy
}

func newDecisionMaker() *decisionMaker {
	dm := &decisionMaker{}
	dm.strategy = &defaultStrategy{}

	return dm
}
