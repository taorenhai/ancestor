package config

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/larspensjo/config"
)

const (
	// SectionDefault is the config file default section
	SectionDefault = "Default"
)

// Config the config object
type Config struct {
	config *config.Config
}

// LoadConfig load the config file
func LoadConfig(configFilePath string) (*Config, error) {
	localConfig := new(Config)

	cfg, err := config.ReadDefault(configFilePath)
	if err != nil {
		return nil, errors.Errorf("Fail to find config, path: %s err: %s", configFilePath, err)
	}

	localConfig.config = cfg
	return localConfig, nil
}

// GetBool get bool value from section and key
func (cfg *Config) GetBool(section, key string) bool {
	if cfg.config.HasSection(section) {
		val, err := cfg.config.Bool(section, key)
		if err != nil {
			panic(fmt.Sprintf("get config bool value failure, section:%v key:%v err:%v", section, key, err))
		}
		return val
	}
	panicSection(section)
	return false
}

// GetFloat get float value from section and key
func (cfg *Config) GetFloat(section, key string) float64 {
	if cfg.config.HasSection(section) {
		val, err := cfg.config.Float(section, key)
		if err != nil {
			panic(fmt.Sprintf("get config float64 value failure, section:%v key:%v err:%v", section, key, err))
		}
		return val
	}

	panicSection(section)
	return 0.0
}

// GetInt get int value from section and key
func (cfg *Config) GetInt(section, key string) int {
	if cfg.config.HasSection(section) {
		val, err := cfg.config.Int(section, key)
		if err != nil {
			panic(fmt.Sprintf("get config int value failure, section:%v key:%v err:%v", section, key, err))
		}
		return val
	}

	panicSection(section)
	return 0
}

// GetString get string value from section and key
func (cfg *Config) GetString(section, key string) string {
	if cfg.config.HasSection(section) {
		val, err := cfg.config.String(section, key)
		if err != nil {
			panic(fmt.Sprintf("get config string value failure, section:%v key:%v err:%v", section, key, err))
		}
		return val
	}

	panicSection(section)
	return ""
}

func panicSection(section string) {
	panic(fmt.Sprintf("the section not exist, section:%v", section))
}
