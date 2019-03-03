package main

import (
	"flag"
	"github.com/ashwinikd/json-log-exporter/collector"
	"github.com/ashwinikd/json-log-exporter/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
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
		log.Panic(err)
	}

	for _, l := range cfg.Logs {
		log.Printf("Initializing Log '%s'", l.Name)
		collector := collector.NewCollector(l)
		collector.Run()
	}

	http.Handle(metricPath, promhttp.Handler())
	http.ListenAndServe(bind, nil)
}