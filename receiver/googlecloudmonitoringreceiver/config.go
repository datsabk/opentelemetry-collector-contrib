// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudmonitoringreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver"

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`

	Region             string    `mapstructure:"region"`
	ProjectID          string    `mapstructure:"project_id"`
	ServiceAccountKey  string    `mapstructure:"service_account_key"`
	CredentialFilePath string    `mapstructure:"credential_file_path"`
	Services           []Service `mapstructure:"services"`
}

type Service struct {
	ServiceName string        `mapstructure:"service_name"`
	Delay       int           `mapstructure:"delay"`
	MetricName  string        `mapstructure:"metric_name"`
	Interval    time.Duration `mapstructure:"interval"`
}

func (config *Config) Validate() error {
	if len(config.Services) == 0 {
		return errors.New("missing required field \"services\" or its value is empty")
	}

	for _, service := range config.Services {
		if err := service.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (service Service) Validate() error {
	if service.ServiceName == "" {
		return errors.New("field \"service_name\" is required and cannot be empty for service configuration")
	}

	if service.Delay < 0 {
		return errors.New("field \"delay\" cannot be negative for service configuration")
	}

	return nil
}
