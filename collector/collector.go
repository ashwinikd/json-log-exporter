package collector

import (
	"bytes"
	"encoding/json"
	"github.com/ashwinikd/json-log-exporter/config"
	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"os"
	"strconv"
	"text/template"
)

type Collector struct {
	Name string
	counters []*counter
	gauges []*gauge
	histograms []*histogram
	summaries []*summary

	cfg *config.LogConfig
}

type counter struct {
	key string
	valueTpl *template.Template
	metric *prometheus.CounterVec
	labelNames []string
	labelValues []*template.Template
	cfg *config.MetricConfig
}

type gauge struct {
	key string
	valueTpl *template.Template
	metric *prometheus.GaugeVec
	labelNames []string
	labelValues []*template.Template
	cfg *config.MetricConfig
}

type histogram struct {
	key string
	valueTpl *template.Template
	metric *prometheus.HistogramVec
	labelNames []string
	labelValues []*template.Template
	cfg *config.MetricConfig
}

type summary struct {
	key string
	valueTpl *template.Template
	metric *prometheus.SummaryVec
	labelNames []string
	labelValues []*template.Template
	cfg *config.MetricConfig
}

func NewCollector(cfg *config.LogConfig) *Collector {
	globalLabels, globalValues := cfg.Labels()
	numCounter := 0
	numGauge := 0
	numHistogram :=0
	numSummary := 0

	for _, metric := range cfg.Metrics {
		if metric.Type == "counter" {
			numCounter++
		} else if metric.Type == "gauge" {
			numGauge++
		} else if metric.Type == "histogram" {
			numHistogram++
		} else if metric.Type == "summary" {
			numSummary++
		} else {
			log.Infof("Found invalid Metric Type [%s]", metric.Type)
		}
	}

	collector := &Collector{
		Name: cfg.Name,
		counters: make([]*counter, numCounter),
		gauges: make([]*gauge, numGauge),
		histograms: make([]*histogram, numHistogram),
		summaries: make([]*summary, numSummary),
		cfg: cfg,
	}

	for _, metric := range cfg.Metrics {
		l, v := metric.Labels()
		var labels []string
		var values []*template.Template
		for i, ln := range globalLabels {
			if ! contains(l, ln) {
				t, err := template.New(cfg.Name + ":" + metric.Name + ":+label_" + ln).Parse(globalValues[i])
				if err == nil {
					values = append(values, t)
					labels = append(labels, ln)
				}
			}
		}
		for i, ln := range l {
			t, err := template.New(cfg.Name + ":" + metric.Name + ":+label_" + ln).Parse(v[i])
			if err == nil {
				labels = append(labels, ln)
				values = append(values, t)
			}
		}

		if metric.Type == "counter" {
			numCounter--
			m := prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: "jsonlog",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
			}, labels)

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			collector.counters[numCounter] = &counter{
				key: metric.ValueKey,
				valueTpl: tpl,
				metric: m,
				labelNames: labels,
				labelValues: values,
				cfg: metric,
			}
		} else if metric.Type == "gauge" {
			numGauge--
			m := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "jsonlog",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
			}, labels)

			if metric.ValueKey == "" {
				log.Fatalf("No key provided for gauge value. [%s.%s]", cfg.Name, metric.Name)
				os.Exit(1)
			}

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			collector.gauges[numGauge] = &gauge{
				key: metric.ValueKey,
				valueTpl: tpl,
				metric: m,
				labelNames: labels,
				labelValues: values,
				cfg: metric,
			}
		} else if metric.Type == "histogram" {
			numHistogram--

			if len(metric.Buckets) == 0 {
				metric.Buckets = prometheus.DefBuckets
			}

			m := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "jsonlog",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
				Buckets: metric.Buckets,
			}, labels)

			if metric.ValueKey == "" {
				log.Fatalf("No key provided for histogram value. [%s.%s]", cfg.Name, metric.Name)
				os.Exit(1)
			}

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			collector.histograms[numHistogram] = &histogram{
				key: metric.ValueKey,
				valueTpl: tpl,
				metric: m,
				labelNames: labels,
				labelValues: values,
				cfg: metric,
			}
		} else if metric.Type == "summary" {
			numSummary--

			if metric.SummaryMaxAge == 0 {
				metric.SummaryMaxAge = prometheus.DefMaxAge
			}

			if metric.SummaryAgeBuckets == 0 {
				metric.SummaryAgeBuckets = prometheus.DefAgeBuckets
			}

			m := prometheus.NewSummaryVec(prometheus.SummaryOpts{
				Namespace: "jsonlog",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
				Objectives: metric.Objectives,
				MaxAge: metric.SummaryMaxAge,
				AgeBuckets: metric.SummaryAgeBuckets,
			}, labels)

			if metric.ValueKey == "" {
				log.Fatalf("No key provided for summary value. [%s.%s]", cfg.Name, metric.Name)
				os.Exit(1)
			}

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			collector.summaries[numSummary] = &summary{
				key: metric.ValueKey,
				valueTpl: tpl,
				metric: m,
				labelNames: labels,
				labelValues: values,
				cfg: metric,
			}
		} else {
			log.Fatalf("Found invalid metric type '%s'", metric.Type)
			os.Exit(1)
		}
	}

	return collector
}

func (this *Collector) Run() {
	this.registerMetrics()

	for _, f := range this.cfg.SourceFiles {
		t, err := tail.TailFile(f, tail.Config{
			Follow: true,
			ReOpen: true,
			Poll: true,
			MustExist: true,
		})

		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		go func() {
			for line := range t.Lines {
				b := []byte(line.Text)
				var data interface{}
				err := json.Unmarshal(b, &data)

				if err != nil {
					log.Warnf("Error in parsing line in file [%s] '%s' | Error => %s", f, line.Text, err)
					continue
				}

				for _, m := range this.counters {
					values := labelValues(data, m.labelValues)
					inc := 1.0

					if m.key != "" {
						vstr := executeTpl(m.valueTpl, data)
						if i, err := strconv.ParseFloat(vstr, 64); err == nil {
							inc = i
							m.metric.WithLabelValues(values...).Add(inc)
						} else {
							log.Warnf("Value for counter is invalid [%s]. Ignoring line.", vstr)
						}
					} else {
						m.metric.WithLabelValues(values...).Add(inc)
					}
				}

				for _, m := range this.gauges {
					values := labelValues(data, m.labelValues)
					vstr := executeTpl(m.valueTpl, data)
					if i, err := strconv.ParseFloat(vstr, 64); err == nil {
						m.metric.WithLabelValues(values...).Add(i)
					} else {
						log.Warnf("Value for gauge is invalid [%s]. Ignoring line.", vstr)
					}
				}

				for _, m := range this.histograms {
					values := labelValues(data, m.labelValues)
					vstr := executeTpl(m.valueTpl, data)
					if i, err := strconv.ParseFloat(vstr, 64); err == nil {
						m.metric.WithLabelValues(values...).Observe(i)
					} else {
						log.Warnf("Value for histogram is invalid [%s]. Ignoring line.", vstr)
					}
				}

				for _, m := range this.summaries {
					values := labelValues(data, m.labelValues)
					vstr := executeTpl(m.valueTpl, data)
					if i, err := strconv.ParseFloat(vstr, 64); err == nil {
						m.metric.WithLabelValues(values...).Observe(i)
					} else {
						log.Warnf("Value for summary is invalid [%s]. Ignoring line.", vstr)
					}
				}
			}
		}()
	}
}

func (this *Collector) registerMetrics() {
	for _, m := range this.counters {
		prometheus.MustRegister(m.metric)
	}
	for _, m := range this.gauges {
		prometheus.MustRegister(m.metric)
	}
	for _, m := range this.histograms {
		prometheus.MustRegister(m.metric)
	}
	for _, m := range this.summaries {
		prometheus.MustRegister(m.metric)
	}
}

func labelValues(f interface{}, templates []*template.Template) (values []string) {
	values = make([]string, len(templates))
	for i, tpl := range templates {
		values[i] = executeTpl(tpl, f)
	}
	return
}

func executeTpl(tpl *template.Template, data interface{}) string {
	var out bytes.Buffer
	err := tpl.Execute(&out, data);
	if err == nil {
		return out.String()
	} else {
		return ""
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}