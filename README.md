# JSON Log Exporter

![Build Status](https://travis-ci.com/ashwinikd/json-log-exporter.svg?branch=master "Travis")

Prometheus exporter for JSON logs, written in Go. This uses 
[hpcloud/tail](https://github.com/hpcloud/tail) for tailing
the files.

## Installation & Usage
Install using following command:
```shell
go get github.com:ashwinikd/json-log-exporter.git
```
Usage:
```
Usage of json-log-exporter:
  -config-file string
    	Configuration file. (default "json_log_exporter.yml")
  -web.listen-address string
    	Address to listen on for the web interface. (default ":9321")
  -web.telemetry-path string
    	Path under which to expose Prometheus metrics. (default "/metrics")
```

##  Configuration
The exporter needs to be configured to be of any use. The configuration file is written
in yaml. Configuration is list of Log groups. Each log group corresponds to possibly
multiple files of similar format. Each log group can be configured as follows:

> Note: `name` and `metric.name` are used to compute the metric name that is reported to
prometheus. So the values for these must comply with 
[Prometheus Metric Names Guidelines](https://prometheus.io/docs/practices/naming/#metric-names).


| Key                 | Type               | Description                 |
|---------------------|--------------------|-----------------------------|
| `name`              | `string`           | A common group name for tailed files. Used as subsystem value when deriving metric name |
| `source_files`      | `Array`            | List of paths of files to tail |
| `labels`            | `Map`              | List of key value pairs of labels to apply to all the metrics derived from tailing |
| `metrics`           | `Array`            | List of metrics to collect |
| `metrics.name`      | `string`           | Metric name |
| `metrics.type`      | `string`           | One of `counter`, `gauge`, `histogram` or `summary` |
| `metrics.labels`    | `Map`              | List of key value pairs of labes to apply to this metric |
| `metrics.value`     | `template`         | Value to use for observations. For gauge and counter the computed value is added to metric value |
| `metrics.buckets`   | `Array`            | List of values to use as buckets. Only used for histograms |
| `metrics.objectives`| `Map`              | Map of quantile to error. Only used for summaries. |
| `metrics.max_age`   | `integer`          | Maximum age of bucket. Only used for summaries. |
| `metrics.age_buckets`| `integer`         | Number of buckets to keep. Only used for summaries. |

### Templating
Go templating language can be used for interpolating values from
parsed log line. This can be used for label values and value field
of metrics. 

### Example
Following is an example configuration
```yaml
- name: requests
  source_files:
    - /var/log/thread1.log
    - /var/log/thread2.log
  labels:
    foo: bar
    user_agent: "{{.user_agent}}"
    client_ip: "{{.x_forwarded_for}}"
  metrics:
    - name: count_total
      type: counter
      labels:
        referer: "{{.referer}}"
    - name: response_bytes_total
      type: counter
      value: "{{.response_bytes}}"
      labels:
        foo: override
    - name: response_time_seconds
      type: histogram
      value: "{{.response_time}}"
      buckets:
        - 0.001
        - 0.05
        - 0.1
      labels:
        key1: value1
    - name: remaining_credits
      type: gauge
      value: "{{.credits}}"
    - name: request_time_seconds
      type: summary
      value: "{{.request_time}}"
      objectives:
        0.5: 0.05
      max_age: 600
      age_buckets: 10
- name: actions
  source_files:
    - /var/log/actions.log
  labels:
    domain: "{{.domain}}"
    actor: "{{.user_id}}"
  metrics:
    - name: count_total
      type: counter
```

## Metric Names

Metric names are derived using following scheme:

```
<namespace>_<log_group>_<metric_name>
```

The namespace used is `jsonlog`. The `<log_group>` is replaced with the name of Log group to which the metric belongs.
`<metric_name>` is substituted with the metric name directly. For example the `count_total` counter of `requests` log 
group in example config will be reported as `jsonlog_requests_count_total`.

## ToDo
1. Store the position of last line read from log file. Right now
exporter will read the log file from beginning on restart.