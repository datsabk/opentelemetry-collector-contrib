package googlecloudmonitoringreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type monitoringReceiver struct {
	config *Config
	logger *zap.Logger
	cancel context.CancelFunc
}

func newGoogleCloudMonitoringReceiver(cfg *Config, logger *zap.Logger) *monitoringReceiver {
	return &monitoringReceiver{
		config: cfg,
		logger: logger,
	}
}

func (m *monitoringReceiver) Start(ctx context.Context, _ component.Host) error {
	ctx, m.cancel = context.WithCancel(ctx)
	err := m.initialize(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (m *monitoringReceiver) Shutdown(context.Context) error {
	m.cancel()
	m.logger.Debug("shutting down googlecloudmonitoringreceiver receiver")

	return nil
}

func (m *monitoringReceiver) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()
	m.logger.Debug("Scrape metrics ")

	return metrics, nil
}

func (m *monitoringReceiver) initialize(ctx context.Context) error {
	m.logger.Debug("Initialize googlecloudmonitoringreceiver receiver from start")

	return nil
}
