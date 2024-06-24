// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudmonitoringreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	minCollectionIntervalSeconds = 60
)

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	TopMetricsQueryMaxRows         int       `mapstructure:"top_metrics_query_max_rows"`
	Region                         string    `mapstructure:"region"`
	ProjectID                      string    `mapstructure:"project_id"`
	ServiceAccountKey              string    `mapstructure:"service_account_key"`
	CredentialFilePath             string    `mapstructure:"credential_file_path"`
	Services                       []Service `mapstructure:"services"`
}

type Service struct {
	ServiceName          string       `mapstructure:"service_name"`
	Delay                int          `mapstructure:"delay"`
	Filters              Filters      `mapstructure:"filters"`
	Interval             TimeInterval `mapstructure:"interval"`
	Aggregation          Aggregation  `mapstructure:"aggregation"`
	SecondaryAggregation Aggregation  `mapstructure:"secondary_aggregation"`
	OrderBy              string       `mapstructure:"order_by"`
	View                 string       `mapstructure:"view"`
	PageSize             int          `mapstructure:"page_size"`
	PageToken            string       `mapstructure:"page_token"`
}

type Filters struct {
	GroupID              string      `mapstructure:"group_id"`
	MetricName           string      `mapstructure:"metric_name"`
	MetricType           []LabelPair `mapstructure:"metric_type"`
	MetricLabels         []LabelPair `mapstructure:"metric_labels"`
	ResourceType         string      `mapstructure:"resource_type"`
	ResourceLabels       []LabelPair `mapstructure:"resource_labels"`
	MetadataSystemLabels []LabelPair `mapstructure:"metadata_system_labels"`
}

type LabelPair struct {
	Name  string `mapstructure:"labels_name"`
	Value string `mapstructure:"labels_value"`
}

type TimeInterval struct {
	StartTime string `mapstructure:"start_time"`
	EndTime   string `mapstructure:"end_time"`
}

type Aggregation struct {
	AlignmentPeriod    string   `mapstructure:"alignmentPeriod"`
	PerSeriesAligner   string   `mapstructure:"perSeriesAligner"`
	CrossSeriesReducer string   `mapstructure:"crossSeriesReducer"`
	GroupByFields      []string `mapstructure:"groupByFields"`
}

func (config *Config) Validate() error {
	if config.CollectionInterval.Seconds() < minCollectionIntervalSeconds {
		return fmt.Errorf("\"collection_interval\" must be not lower than %v seconds, current value is %v seconds", minCollectionIntervalSeconds, config.CollectionInterval.Seconds())
	}

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

	if err := service.Filters.Validate(); err != nil {
		return err
	}

	return nil
}

func (filters Filters) Validate() error {
	if filters.GroupID == "" {
		return errors.New("field \"group_id\" is required and cannot be empty for filters configuration")
	}

	if filters.MetricName == "" {
		return errors.New("field \"metric_name\" is required and cannot be empty for filters configuration")
	}

	for _, label := range filters.MetricLabels {
		if err := label.Validate(); err != nil {
			return err
		}
	}

	for _, label := range filters.ResourceLabels {
		if err := label.Validate(); err != nil {
			return err
		}
	}

	for _, label := range filters.MetadataSystemLabels {
		if err := label.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (label LabelPair) Validate() error {
	if label.Name == "" {
		return errors.New("field \"labels_name\" is required and cannot be empty for label pair configuration")
	}

	if label.Value == "" {
		return errors.New("field \"labels_value\" is required and cannot be empty for label pair configuration")
	}

	return nil
}
