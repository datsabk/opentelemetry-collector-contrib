package googlecloudmonitoringreceiver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
)

type monitoringReceiver struct {
	config    *Config
	logger    *zap.Logger
	cancel    context.CancelFunc
	client    *monitoring.MetricClient
	startOnce sync.Once
}

func newGoogleCloudMonitoringReceiver(cfg *Config, logger *zap.Logger) *monitoringReceiver {
	return &monitoringReceiver{
		config: cfg,
		logger: logger,
	}
}

func (m *monitoringReceiver) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()
	m.logger.Debug("Scrape metrics ")

	return metrics, nil
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

func (m *monitoringReceiver) initialize(ctx context.Context) error {
	servicePath := m.config.ServiceAccountKey

	m.startOnce.Do(func() {
		client, err := monitoring.NewMetricClient(ctx, option.WithCredentialsFile(servicePath))
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		m.client = client
	})

	ctx, m.cancel = context.WithCancel(ctx)

	// API for collect metrics data from timseries API
	_, err := m.collectMetricsDataFromEndpoint(ctx)
	if err != nil {
		return err
	}
	return nil
}

func calculateStartEndTime(interval, delay time.Duration) (time.Time, time.Time) {
	// Get the current time
	now := time.Now()

	// Calculate the start time (current time + delay)
	startTime := now.Add(delay)

	// Calculate the end time (start time + interval)
	endTime := startTime.Add(interval)

	return startTime, endTime
}

func (m *monitoringReceiver) collectMetricsDataFromEndpoint(ctx context.Context) (*monitoringpb.TimeSeries, error) {
	var calStartTime time.Time
	var calEndTime time.Time

	var filterQuery string
	for _, service := range m.config.Services {
		// Define the interval and delay times
		interval := service.Interval
		delay := service.Delay

		calStartTime, calEndTime = calculateStartEndTime(interval, delay)

		if service.ServiceName == "compute" {
			if service.MetricName != "" {
				filterQuery = service.MetricName
			} else {
				filterQuery = `metric.type = "compute.googleapis.com/instance/cpu/usage_time"`
			}
		}
	}

	// Define the request
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + m.config.ProjectID,
		Filter: filterQuery,
		Interval: &monitoringpb.TimeInterval{
			EndTime:   &timestamppb.Timestamp{Seconds: calEndTime.Unix()},
			StartTime: &timestamppb.Timestamp{Seconds: calStartTime.Unix()},
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}

	it := m.client.ListTimeSeries(ctx, req)
	fmt.Println("Time series data:")

	var resp *monitoringpb.TimeSeries
	for {
		resp, err := it.Next()
		fmt.Println("\n resp", resp)
		if err != nil {
			if err.Error() == "iterator: Done" {
				break
			}

			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, fmt.Errorf("failed to retrieve time series data: %v", err)
		}
	}
	return resp, nil
}

