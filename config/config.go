package config

import (
	"errors"

	"github.com/cyverse-de/go-mod/cfg"
)

var ServiceName = "qms"

// Specification defines the configuration settings for the qms service.
type Specification struct {
	DatabaseURI         string
	RunSchemaMigrations bool
	ReinitDB            bool
	DotEnvPath          string
	ConfigPath          string
	EnvPrefix           string
	UsernameSuffix      string
}

// LoadConfig loads the configuration for the qms service.
func LoadConfig(envPrefix, configPath, dotEnvPath string) (*Specification, error) {
	k, err := cfg.Init(&cfg.Settings{
		EnvPrefix:   envPrefix,
		ConfigPath:  configPath,
		DotEnvPath:  dotEnvPath,
		StrictMerge: false,
		FileType:    cfg.YAML,
	})
	if err != nil {
		return nil, err
	}

	var s Specification

	s.DatabaseURI = k.String("database.uri")
	if s.DatabaseURI == "" {
		return nil, errors.New("database.uri or QMS_DATABASE_URI must be set")
	}

	s.RunSchemaMigrations = k.Bool("database.migrate")
	s.ReinitDB = k.Bool("database.reinit")

	s.UsernameSuffix = k.String("username.suffix")
	if s.UsernameSuffix == "" {
		return nil, errors.New("username.suffix or QMS_USERNAME_SUFFIX must be set")
	}

	return &s, err
}
