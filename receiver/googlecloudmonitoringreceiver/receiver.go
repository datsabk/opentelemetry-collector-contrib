package googlecloudmonitoringreceiver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

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
			return
		}

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

// collectMetricsDataFromEndpoint collects metrics data from the specified endpoint based on the configuration.
func (m *monitoringReceiver) collectMetricsDataFromEndpoint(ctx context.Context) (*monitoringpb.TimeSeries, error) {
	var calStartTime time.Time
	var calEndTime time.Time
	var filterQuery string

	// Iterate over each service in the configuration to calculate start/end times and construct the filter query.
	for _, service := range m.config.Services {
		// Define the interval and delay times
		interval := service.Interval
		delay := service.Delay

		// Calculate the start and end times
		calStartTime, calEndTime = calculateStartEndTime(interval, delay)

		// Get the filter query for the service
		filterQuery = getFilterQuery(service)

		// Log an error if the filter query is empty
		if filterQuery == "" {
			m.logger.Error("Internal Server Error")
		}
	}

	// Define the request to list time series data
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + m.config.ProjectID,
		Filter: filterQuery,
		Interval: &monitoringpb.TimeInterval{
			EndTime:   &timestamppb.Timestamp{Seconds: calEndTime.Unix()},
			StartTime: &timestamppb.Timestamp{Seconds: calStartTime.Unix()},
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}

	// Create an iterator for the time series data
	it := m.client.ListTimeSeries(ctx, req)
	m.logger.Info("Time series data:")

	var resp *monitoringpb.TimeSeries

	// Iterate over the time series data
	for {
		resp, err := it.Next()
		respData := fmt.Sprintf("\n\nresp => %s", resp)
		m.logger.Info(respData)

		// Handle errors and break conditions for the iterator
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

// calculateStartEndTime calculates the start and end times based on the current time, interval, and delay.
func calculateStartEndTime(interval, delay time.Duration) (time.Time, time.Time) {
	// Get the current time
	now := time.Now()

	// Calculate the start time (current time - delay)
	startTime := now.Add(-delay)

	// Calculate the end time (start time + interval)
	endTime := startTime.Add(interval)

	return startTime, endTime
}

// getFilterQuery constructs a filter query string based on the provided service.
func getFilterQuery(service Service) string {
	var filterQuery string
	const baseQuery = `metric.type =`
	const defaultComputeMetric = "compute.googleapis.com/instance/cpu/usage_time"

	switch service.ServiceName {
	case "compute":
		if service.MetricName != "" {
			// If a specific metric name is provided, use it in the filter query
			filterQuery = fmt.Sprintf(`%s "%s"`, baseQuery, service.MetricName)
			return filterQuery
		} else {
			// If no specific metric name is provided, use the default compute metric
			filterQuery = fmt.Sprintf(`%s "%s"`, baseQuery, defaultComputeMetric)
			return filterQuery
		}
		// Add other service cases here
	default:
		// Return an empty string if the service is not recognized
		return ""
	}
}
