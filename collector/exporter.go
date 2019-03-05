package collector

import (
	"github.com/ashwinikd/json-log-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var exporters = make(map[string]*Exporter)

type Exporter struct {
	Name string
	Registry prometheus.Registerer
	Handler http.Handler
	config *config.ExportConfig
}

func InitializeExports(exports []*config.ExportConfig) {
	for _, cfg := range exports {
		registry := prometheus.NewRegistry()
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		exporter := &Exporter{
			Name: cfg.Name,
			Registry: registry,
			Handler: handler,
			config: cfg,
		}
		exporters[cfg.Name] = exporter
	}
}

func GetExport(name string) *Exporter{
	return exporters[name]
}