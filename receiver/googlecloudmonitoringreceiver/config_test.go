// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudmonitoringreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver/internal/metadata"
)

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(component.NewIDWithName(metadata.Type, "").String())
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(cfg))

	assert.Equal(t,
		&Config{
			ControllerConfig: scraperhelper.ControllerConfig{
				CollectionInterval: 120 * time.Second,
			},
			Region:            "us-central1",
			ProjectID:         "my-project-id",
			ServiceAccountKey: "path/to/service_account.json",
			Services: []Service{
				{
					ServiceName: "compute",
					Delay:       60 * time.Second,
					MetricName:  "compute.googleapis.com/instance/cpu/usage_time",
				},
			},
		},
		cfg,
	)
}

func TestValidateService(t *testing.T) {
	testCases := map[string]struct {
		service      Service
		requireError bool
	}{
		"Valid Service": {
			Service{
				ServiceName: "service_name",
				Delay:       0,
			}, false},
		"Empty ServiceName": {
			Service{
				ServiceName: "",
				Delay:       0,
			}, true},
		"Negative Delay": {
			Service{
				ServiceName: "service_name",
				Delay:       -1,
			}, true},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			err := testCase.service.Validate()
			if testCase.requireError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	validService := Service{
		ServiceName: "compute",
		Delay:       60 * time.Second, // Ensure delay is valid
		MetricName:  "compute.googleapis.com/instance/cpu/usage_time",
	}

	testCases := map[string]struct {
		collectionInterval time.Duration
		services           []Service
		requireError       bool
	}{
		"Valid Config":                {120 * time.Second, []Service{validService}, false},
		"Invalid Collection Interval": {0, []Service{validService}, true},
		"Empty Services":              {120 * time.Second, nil, true},
		"Invalid Service in Services": {120 * time.Second, []Service{{}}, true},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{
				ControllerConfig: scraperhelper.ControllerConfig{
					CollectionInterval: testCase.collectionInterval,
				},
				Services: testCase.services,
			}

			err := cfg.Validate()
			if testCase.requireError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
