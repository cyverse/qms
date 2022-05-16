package config

import "github.com/kelseyhightower/envconfig"

var ServiceName = "QMS"

// Specification defines the configuration settings for the QMS service.
type Specification struct {
	DatabaseURI string `required:"true" split_words:"true"`
	ReinitDB    bool   `default:"false" split_words:"true"`
}

// LoadConfig loads the configuration for the QMS service.
func LoadConfig() (*Specification, error) {
	var s Specification
	err := envconfig.Process("qms", &s)
	return &s, err
}
