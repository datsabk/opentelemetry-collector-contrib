package googlecloudmonitoringreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver"

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudmonitoringreceiver/internal"

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
		interval := m.config.CollectionInterval
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
		respData := fmt.Sprintf("\n \n resp => %s \n \n", resp)
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

		// Convert the GCP TimeSeries to pmetric.Metrics format of OpenTelemetry
		metrics := convertGCPTimeSeriesToMetrics(resp)

		// Process or export the metrics as needed
		dataPointsCount := fmt.Sprintf("\n \n Converted metrics: %+v \n \n ", metrics.DataPointCount())
		resourceMetrics := fmt.Sprintf("\n \n Converted metrics: %+v \n \n", metrics.ResourceMetrics())
		m.logger.Info(dataPointsCount)
		m.logger.Info(resourceMetrics)
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

// ConvertGCPTimeSeriesToMetrics converts GCP Monitoring TimeSeries to pmetric.Metrics
func convertGCPTimeSeriesToMetrics(resp *monitoringpb.TimeSeries) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()

	// Set metric name and description
	m.SetName(resp.Metric.Type)
	m.SetUnit(resp.Unit)

	// Assuming MetricDescriptor and description are set
	m.SetDescription("Converted from GCP Monitoring TimeSeries")

	// Set resource labels
	resource := rm.Resource()
	resource.Attributes().PutStr("resource_type", resp.Resource.Type)
	for k, v := range resp.Resource.Labels {
		resource.Attributes().PutStr(k, v)
	}

	// Set metadata (user and system labels)
	if resp.Metadata != nil {
		for k, v := range resp.Metadata.UserLabels {
			resource.Attributes().PutStr(k, v)
		}
		if resp.Metadata.SystemLabels != nil {
			for k, v := range resp.Metadata.SystemLabels.Fields {
				resource.Attributes().PutStr(k, fmt.Sprintf("%v", v))
			}
		}
	}

	switch resp.GetMetricKind() {
	case metric.MetricDescriptor_GAUGE:
		internal.ConvertGaugeToMetrics(resp, m)
	case metric.MetricDescriptor_CUMULATIVE:
		internal.ConvertSumToMetrics(resp, m)
	case metric.MetricDescriptor_DELTA:
		internal.ConvertDeltaToMetrics(resp, m)
	// Add cases for SUMMARY, HISTOGRAM, EXPONENTIAL_HISTOGRAM if needed
	default:
		log.Printf("Unsupported metric kind: %v\n", resp.GetMetricKind())
	}

	return metrics
}

