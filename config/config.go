package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

type Config struct {
	Labels map[string]string `yaml:"labels"`
	LogGroups []LogGroupConfig `yaml:"log_groups"`
	Exports []*ExportConfig `yaml:"exports"`
	original  string
}

type ExportConfig struct {
	Name string `yaml:"name"`
	MetricPath string `yaml:"path"`
}

type LogGroupConfig struct {
	Name string `yaml:"name"`
	SourceFiles []string `yaml:"files"`
	GlobalLabels  map[string]string `yaml:"labels"`
	Metrics []*MetricConfig `yaml:"metrics"`
}

type MetricConfig struct {
	Name string `yaml:"name"`
	Desc string `yaml:"help"`
	Type string `yaml:"type"`
	Export string `yaml:"export_to"`
	Buckets []float64 `yaml:"buckets"`
	Objectives map[float64]float64 `yaml:"objectives"`
	SummaryMaxAge time.Duration `yaml:"max_age"`
	SummaryAgeBuckets uint32 `yaml:"age_buckets"`
	MetricLabels  map[string]string `yaml:"labels"`
	ValueKey string `yaml:"value"`
}

func (this *LogGroupConfig) Labels() (labels, values []string) {
	labels = make([]string, len(this.GlobalLabels))
	values = make([]string, len(this.GlobalLabels))

	i := 0
	for k, v := range this.GlobalLabels {
		labels[i] = k
		values[i] = v
		i++
	}

	return
}

func (this *MetricConfig) Labels() (labels, values []string) {
	labels = make([]string, len(this.MetricLabels))
	values = make([]string, len(this.MetricLabels))

	i := 0
	for k, v := range this.MetricLabels {
		labels[i] = k
		values[i] = v
		i++
	}

	return
}

func LoadFile(filename string) (conf *Config, err error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	conf, err = load(string(content))
	return
}

func load(s string) (*Config, error) {
	var (
		cfg  = &Config{}
	)

	err := yaml.Unmarshal([]byte(s), &cfg)
	if err != nil {
		return nil, err
	}

	cfg.original = s
	cfg.prepareLabels()

	return cfg, nil
}

func (this *Config) prepareLabels() {
	for _, logGroups := range this.LogGroups {
		for ln, lv := range this.Labels {
			if _, exists := logGroups.GlobalLabels[ln]; ! exists {
				logGroups.GlobalLabels[ln] = lv
			}
		}
	}
}
