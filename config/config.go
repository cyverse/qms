package config

import (
	"errors"

	"github.com/cyverse-de/go-mod/cfg"
)

var ServiceName = "QMS"

// Specification defines the configuration settings for the QMS service.
type Specification struct {
	DatabaseURI   string
	ReinitDB      bool
	NatsCluster   string
	DotEnvPath    string
	ConfigPath    string
	EnvPrefix     string
	MaxReconnects int
	ReconnectWait int
	CACertPath    string
	TLSKeyPath    string
	TLSCertPath   string
	CredsPath     string
	BaseSubject   string
	BaseQueueName string
}

// LoadConfig loads the configuration for the QMS service.
func LoadConfig(envPrefix, configPath, dotEnvPath string) (*Specification, error) {
	k, err := cfg.Init(&cfg.Settings{
		EnvPrefix:   envPrefix,
		ConfigPath:  configPath,
		DotEnvPath:  dotEnvPath,
		StrictMerge: false,
		FileType:    cfg.YAML,
	})

	var s Specification

	s.DatabaseURI = k.String("database.uri")
	if s.DatabaseURI == "" {
		return nil, errors.New("database.uri or QMS_DATABASE_URI must be set")
	}

	s.ReinitDB = k.Bool("reinit.db")

	s.NatsCluster = k.String("nats.cluster")
	if s.NatsCluster == "" {
		return nil, errors.New("nats.cluster must be set in the configuration file")
	}

	return &s, err
}
