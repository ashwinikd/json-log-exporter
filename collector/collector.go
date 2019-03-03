package collector

import (
	"bytes"
	"encoding/json"
	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ashwinikd/json-log-exporter/config"
	"log"
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
			log.Fatalf("Found invalid Metric Type [%s]", metric.Type)
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
				Namespace: "jsonfile",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
			}, labels)

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Panic(err)
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
				Namespace: "jsonfile",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
			}, labels)

			if metric.ValueKey == "" {
				log.Panic("No key provided for gauge value.")
			}

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Panic(err)
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
				Namespace: "jsonfile",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
				Buckets: metric.Buckets,
			}, labels)

			if metric.ValueKey == "" {
				log.Panic("No key provided for histogram value.")
			}

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Panic(err)
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
				Namespace: "jsonfile",
				Subsystem: cfg.Name,
				Name: metric.Name,
				Help: metric.Desc,
				Objectives: metric.Objectives,
				MaxAge: metric.SummaryMaxAge,
				AgeBuckets: metric.SummaryAgeBuckets,
			}, labels)

			if metric.ValueKey == "" {
				log.Panic("No key provided for summary value.")
			}

			tpl, err := template.New(cfg.Name + ":" + metric.Name + ":+value").Parse(metric.ValueKey)
			if err != nil {
				log.Panic(err)
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
			log.Fatalf("Found invalid Metric Type [%s]", metric.Type)
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
			log.Panic(err)
		}

		go func() {
			for line := range t.Lines {
				b := []byte(line.Text)
				var f interface{}
				err := json.Unmarshal(b, &f)

				if err != nil {
					log.Print(err)
					continue
				}

				for _, m := range this.counters {
					values := labelValues(f, m.labelValues)
					inc := 1.0

					if m.key != "" {
						vstr := executeTpl(m.valueTpl, f)
						if i, err := strconv.ParseFloat(vstr, 64); err == nil {
							inc = i
						}
					}
					m.metric.WithLabelValues(values...).Add(inc)
				}

				for _, m := range this.gauges {
					values := labelValues(f, m.labelValues)
					vstr := executeTpl(m.valueTpl, f)
					if i, err := strconv.ParseFloat(vstr, 64); err == nil {
						m.metric.WithLabelValues(values...).Add(i)
					}
				}

				for _, m := range this.histograms {
					values := labelValues(f, m.labelValues)
					vstr := executeTpl(m.valueTpl, f)
					if i, err := strconv.ParseFloat(vstr, 64); err == nil {
						m.metric.WithLabelValues(values...).Observe(i)
					}
				}

				for _, m := range this.summaries {
					values := labelValues(f, m.labelValues)
					vstr := executeTpl(m.valueTpl, f)
					if i, err := strconv.ParseFloat(vstr, 64); err == nil {
						m.metric.WithLabelValues(values...).Observe(i)
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