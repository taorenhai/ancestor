package meta

import "sync"

// RangeStats is mainly used to statics and report to pd server
type RangeStats struct {
	RangeStatsInfo *RangeStatsInfo
	sync.Mutex
}

// UpdateRangeStats update the statics on put operation. Total mainly counts
// all data.
func (rs *RangeStats) UpdateRangeStats(origKeySize, origValSize, keySize, valueSize int) {
	rs.Lock()
	defer rs.Unlock()

	if keySize != 0 || valueSize != 0 {
		rs.RangeStatsInfo.TotalBytes += int64(keySize)
		rs.RangeStatsInfo.TotalBytes += int64(valueSize)
		rs.RangeStatsInfo.TotalCount++
	}

	if origKeySize != 0 || origValSize != 0 {
		rs.RangeStatsInfo.TotalBytes -= int64(origKeySize)
		rs.RangeStatsInfo.TotalBytes -= int64(origValSize)
		rs.RangeStatsInfo.TotalCount--
	}
}
