// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudmonitoringreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver"

import (
	"context"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver/internal/metadata"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability))
}

// createDefaultConfig creates the default exporter configuration
func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig:   scraperhelper.ControllerConfig{},
		Region:             "us-est-1",
		ProjectID:          "",
		ServiceAccountKey:  "",
		CredentialFilePath: "firebase.json",
		Services: []Service{
			{
				ServiceName: "compute",
				Delay:       240,
				Interval:    10,
			},
		},
	}
}

func createMetricsReceiver(
	_ context.Context,
	settings receiver.Settings,
	baseCfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {

	rCfg := baseCfg.(*Config)
	r := newGoogleCloudMonitoringReceiver(rCfg, settings.Logger)

	scraper, err := scraperhelper.NewScraper(metadata.Type.String(), r.Scrape, scraperhelper.WithStart(r.Start),
		scraperhelper.WithShutdown(r.Shutdown))
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(&rCfg.ControllerConfig, settings, consumer,
		scraperhelper.AddScraper(scraper))
}
