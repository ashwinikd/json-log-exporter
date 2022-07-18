# JSON Log Exporter

![Build Status](https://travis-ci.com/ashwinikd/json-log-exporter.svg?branch=master "Travis")

Prometheus exporter for JSON logs, written in Go. This uses 
[hpcloud/tail](https://github.com/hpcloud/tail) for tailing
the files.

## Installation & Usage
Install using following command:
```shell
go install github.com/ashwinikd/json-log-exporter@latest
```
Usage:
```
Usage of json-log-exporter:
  -config-file string
    	Configuration file. (default "json_log_exporter.yml")
  -web.listen-address string
    	Address to listen on for the web interface. (default ":9321")
```

##  Configuration
The exporter needs to be configured to be of any use. The configuration file is written
in yaml. Configuration is list of Global labels, Exports and Log groups. 

The labels specified in the `labels` tags are applied to all the metrics configured. The values
can be overridden at log group level using `log_groups.labels` key or at metric level using `log_groups.metrics.labels`.

`exports` section allows you to split the metrics across different jobs. You need to specify at least 1 export url.

Each log group corresponds to possibly multiple files of similar format. Each log group can be configured as follows:

> Note: `name` and `metric.name` are used to compute the metric name that is reported to
prometheus. So the values for these must comply with 
[Prometheus Metric Names Guidelines](https://prometheus.io/docs/practices/naming/#metric-names).


| Key                   | Type       | Description                                                                                      |
|-----------------------|------------|--------------------------------------------------------------------------------------------------|
| `name`                | `string`   | A common group name for tailed files.                                                            |
| `subsystem`           | `string`   | An optional subsystem name, used for deriving the metric name.                                   |
| `files`               | `Array`    | List of paths of files to tail                                                                   |
| `labels`              | `Map`      | List of key value pairs of labels to apply to all the metrics derived from tailing               |
| `metrics`             | `Array`    | List of metrics to collect                                                                       |
| `metrics.name`        | `string`   | Metric name                                                                                      |
| `metrics.type`        | `string`   | One of `counter`, `gauge`, `histogram` or `summary`                                              |
| `metrics.labels`      | `Map`      | List of key value pairs of labes to apply to this metric                                         |
| `metrics.value`       | `template` | Value to use for observations. For gauge and counter the computed value is added to metric value |
| `metrics.buckets`     | `Array`    | List of values to use as buckets. Only used for histograms                                       |
| `metrics.objectives`  | `Map`      | Map of quantile to error. Only used for summaries.                                               |
| `metrics.max_age`     | `integer`  | Maximum age of bucket. Only used for summaries.                                                  |
| `metrics.age_buckets` | `integer`  | Number of buckets to keep. Only used for summaries.                                              |
| `metrics.export_to`   | `string`   | Name of the exporter from export section                                                         |

### Templating
Go templating language can be used for interpolating values from
parsed log line. This can be used for label values and value field
of metrics. 

### Example
Following is an example configuration
```yaml
namespace: jsonlog
labels:
  foo: bar
exports:
  - name: export1
    path: /metrics/counts-and-gauges
  - name: anotherexport
    path: /metrics/histo-and-summ
log_groups:
  - name: requests
    subsystem: requests
    files:
      - /var/log/thread1.log
      - /var/log/thread2.log
    labels:
      foo: bar
      user_agent: "{{.user_agent}}"
      client_ip: "{{.x_forwarded_for}}"
    metrics:
      - name: count_total
        type: counter
        export_to: export1
        labels:
          referer: "{{.referer}}"
      - name: response_bytes_total
        type: counter
        value: "{{.response_bytes}}"
        export_to: export1
        labels:
          foo: override
      - name: response_time_seconds
        type: histogram
        value: "{{.response_time}}"
        export_to: anotherexport
        buckets:
          - 0.001
          - 0.05
          - 0.1
        labels:
          key1: value1
      - name: remaining_credits
        type: gauge
        value: "{{.credits}}"
        export_to: export1
      - name: request_time_seconds
        type: summary
        value: "{{.request_time}}"
        export_to: anotherexport
        objectives:
          0.5: 0.05
        max_age: 600
        age_buckets: 10
  - name: actions
    subsystem: actions
    files:
      - /var/log/actions.log
    labels:
      domain: "{{.domain}}"
      actor: "{{.user_id}}"
    metrics:
      - name: count_total
        type: counter
        export_to: export1
```

## Metric Names

Metric names are derived using following scheme:

```
<namespace>_<subsystem>_<metric_name>
```

Both `<namespace>` and `<subsystem>` can be left blank optionally. `<metric_name>` is substituted with the metric name directly. For example the `count_total` counter of `requests` log 
group in example config will be reported as `jsonlog_requests_count_total`.

## ToDo
1. Store the position of last line read from log file. Right now
exporter will read the log file from beginning on restart.