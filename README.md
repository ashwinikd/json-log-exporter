# JSON Log Exporter
Prometheus exporter for JSON logs, written in Go. 

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