package util

const (
	// ETCDRootPath etcd root path.
	ETCDRootPath = "ancestor"
)

var (
	// SystemPrefix system key prefix.
	SystemPrefix = []byte("\x00")

	// TxnPrefix for txn to make new key.
	TxnPrefix = []byte("\x01")

	// UserPrefix is the prefix key for data.
	UserPrefix = []byte("\x02")

	// EndPrefix is the max prefix key for UserData.
	EndPrefix = []byte("\x03")

	// HardStatePrefix is raft hard state prefix.
	HardStatePrefix = bytesJoin(SystemPrefix, []byte("rfhs"))

	// AppliedIndexPrefix is raft applied index prefix.
	AppliedIndexPrefix = bytesJoin(SystemPrefix, []byte("rfai"))

	// LastIndexPrefix is raft last index prefix.
	LastIndexPrefix = bytesJoin(SystemPrefix, []byte("rfli"))

	// LogKeyPrefix is raft log prefix.
	LogKeyPrefix = bytesJoin(SystemPrefix, []byte("rflg"))

	// OffsetIndexPrefix is raft offset index prefix.
	OffsetIndexPrefix = bytesJoin(SystemPrefix, []byte("rfof"))

	// RangeStatsKeyPrefix is range statics prefix.
	RangeStatsKeyPrefix = bytesJoin(SystemPrefix, []byte("stat"))
)

// bytesJoin make a new []byte which is the concatenation of the given inputs, in order.
func bytesJoin(keys ...[]byte) []byte {
	var b []byte
	for _, k := range keys {
		b = append(b, k...)
	}
	return b
}
