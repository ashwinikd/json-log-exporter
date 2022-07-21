package main

import (
	"flag"
	"github.com/ashwinikd/json-log-exporter/collector"
	"github.com/ashwinikd/json-log-exporter/config"
	"log"
	"net"
	"net/http"
)

var (
	bind, configFile string
)

func main() {
	flag.StringVar(&bind, "web.listen-address", ":9321", "Address to listen on for the web interface.")
	flag.StringVar(&configFile, "config-file", "json_log_exporter.yml", "Configuration file.")

	flag.Parse()

	cfg, err := config.LoadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	collector.InitializeExports(cfg.Exports)

	for _, logGroup := range cfg.LogGroups {
		log.Printf("Initializing log group '%s'\n", logGroup.Name)
		logGroup := collector.NewCollector(&logGroup, cfg.Namespace)
		logGroup.Run()
	}

	for _, export := range cfg.Exports {
		log.Printf("Exposing '%s'\n", export.MetricPath)
		http.Handle(export.MetricPath, collector.GetExport(export.Name).Handler)
	}

	l, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("HTTP server listening on %s", bind)

	if err := http.Serve(l, nil); err != nil {
		log.Fatal(err)
		log.Fatal(l.Close())
	}
}
