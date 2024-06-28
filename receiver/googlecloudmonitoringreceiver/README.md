# Google Cloud Monitoring Receiver

| Status                   |               |
| ------------------------ | ------------- |
| Stability                | [development] |
| Supported pipeline types | metrics       |
| Distributions            | [contrib]     |

This receiver gets GCP (Google Clout Platform) metrics from [GCP REST API] (https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.timeSeries/list) via the [Google SDK for GCP Metrics](https://cloud.google.com/monitoring/docs/monitoring-overview)

The ultimate goal of Google Cloud Monitoring Receiver is to collect timeseries metrics data for all google services and transform those metrics data into Pipeline Format Data that would be convenient for further uses.

## Configuration

The following configuration options are supported:

```yaml
receivers:
  googlecloudmonitoring:
    collection_interval: 120s
    region: us-central1
    project_id: my-project-id
    service_account_key: "path/to/service_account.json"
    services:
      - service_name: "compute"
        metric_name: "compute.googleapis.com/instance/cpu/usage_time"
        delay: 60s
        interval: 120s
```

- `collection_interval` (Optional): The interval at which metrics are collected. Default is 60s.
- `region` (Required): The GCP region where the services are located.
- `project_id` (Required): The GCP project ID.
- `service_account_key` (Required): The path to the service account key JSON file.
- `services` (Required): A list of services to monitor.

Each service can have the following configuration:

- `service_name` (Required): The name of the GCP service (e.g., `compute`).
- `delay` (Optional): The delay before starting the collection of metrics for this service. Default is 0s.
- `metric_name` (Optional): The specific metric name to collect. If not set, all metrics are collected.
- `interval` (Optional): The interval at which metrics for this service are collected. Default is 240s.

### Filtering

**Metrics Parameters** :

- A monitoring filter that specifies which time series should be returned. The filter must specify a single metric type. For example: `metric_name: "compute.googleapis.com/instance/cpu/usage_time"`

## Sample Configs

[alpha]: https://github.com/open-telemetry/opentelemetry-collector?tab=readme-ov-file#development
[Issue]: https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/2286
