package pd

import (
	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/config"
	"github.com/taorenhai/ancestor/util"
)

const (
	defaultMasterLease         = 3
	defaultTsSaveInterval      = 2000
	defaultCheckInterval       = 30
	defaultRangeCapacity       = 64e6
	defaultRangeSplitThreshold = 0.5
)

const (
	splitTypeManual = iota
	splitTypeAuto
)

// Config is the pd server configuration.
type Config struct {
	// Server listening address.
	Host string

	// Etcd endpoints
	EtcdHosts []string

	// LeaderLease time, if leader doesn't update its TTL
	// in etcd after lease time, etcd will expire the master key
	// and other servers can campaign the master again.
	// Etcd onlys support seoncds TTL, so here is second too.
	MasterLease int64

	// TsSaveInterval is the interval time (ms) to save timestamp.
	TsSaveInterval int64

	// checkInterval is the interval time (second) to check split and migrate.
	CheckInterval int64

	// RangeCapacity
	RangeCapacity uint64

	// RangeSplitThreshold
	RangeSplitThreshold float32

	// RangeSplitType
	RangeSplitType int32

	// WebHost
	WebHost string
	// WebPath
	WebPath string
}

func (c *Config) check() {
	if c.MasterLease <= 0 {
		c.MasterLease = defaultMasterLease
	}

	if c.TsSaveInterval <= 0 {
		c.TsSaveInterval = defaultTsSaveInterval
	}

	if c.CheckInterval <= 0 {
		c.CheckInterval = defaultCheckInterval
	}

	if c.RangeCapacity == 0 {
		c.RangeCapacity = defaultRangeCapacity
	}

	if c.RangeSplitThreshold == 0.0 {
		c.RangeSplitThreshold = defaultRangeSplitThreshold
	}

	if c.RangeSplitType == 0 {
		c.RangeSplitType = splitTypeAuto
	}
}

func parseConfig(filePath string, debug bool, pdCfg *Config) error {
	if !util.IsFileExists(filePath) {
		return errors.New("the config not exist")
	}

	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		return errors.Trace(err)
	}

	pdCfg.Host = cfg.GetString(config.SectionDefault, "Host")
	pdCfg.EtcdHosts = util.SplitString(cfg.GetString(config.SectionDefault, "EtcdHosts"), ";")
	pdCfg.MasterLease = int64(cfg.GetInt(config.SectionDefault, "MasterLease"))
	pdCfg.TsSaveInterval = int64(cfg.GetInt(config.SectionDefault, "TsSaveInterval"))
	pdCfg.CheckInterval = int64(cfg.GetInt(config.SectionDefault, "CheckInterval"))
	pdCfg.RangeCapacity = uint64(cfg.GetInt(config.SectionDefault, "RangeCapacity"))
	pdCfg.RangeSplitThreshold = float32(cfg.GetFloat(config.SectionDefault, "RangeSplitThreshold"))
	pdCfg.RangeSplitType = int32(cfg.GetInt(config.SectionDefault, "RangeSplitType"))

	pdCfg.WebHost = cfg.GetString("Web", "Host")
	pdCfg.WebPath = cfg.GetString("Web", "Path")

	// set log info
	log.SetLevelByString(cfg.GetString("Log", "Level"))
	if !debug {
		log.SetOutputByName(cfg.GetString("Log", "Path"))
		log.SetRotateByDay()
	}

	return nil
}

// NewConfig is used to create a config struct.
func NewConfig(path string, debug bool) (*Config, error) {
	pdCfg := &Config{}
	if err := parseConfig(path, debug, pdCfg); err != nil {
		return nil, errors.Trace(err)
	}

	return pdCfg, nil
}

// LoadConfig is used to load pd server config files.
func (s *Server) LoadConfig(path string) error {
	if err := parseConfig(path, s.debug, s.cfg); err != nil {
		return errors.Trace(err)
	}

	return nil
}
