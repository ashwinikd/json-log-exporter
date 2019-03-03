package main

import (
	"flag"
	"github.com/ashwinikd/json-log-exporter/collector"
	"github.com/ashwinikd/json-log-exporter/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"net"
	"net/http"
)

var (
	bind, configFile, metricPath string
)

func main()  {
	flag.StringVar(&bind, "web.listen-address", ":9321", "Address to listen on for the web interface.")
	flag.StringVar(&configFile, "config-file", "json_log_exporter.yml", "Configuration file.")
	flag.StringVar(&metricPath, "web.telemetry-path", "/metrics", "Path under which to expose Prometheus metrics.")

	flag.Parse()

	cfg, err := config.LoadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, logGroup := range cfg.LogGroups {
		log.Infof("Initializing Log '%s'\n", logGroup.Name)
		logGroup := collector.NewCollector(logGroup)
		logGroup.Run()
	}

	http.Handle(metricPath, promhttp.Handler())

	l, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("HTTP server listening on %s", bind)

	if err := http.Serve(l, nil); err != nil {
		log.Fatal(err)
		log.Fatal(l.Close())
	}
}