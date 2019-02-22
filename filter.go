package main

type filter struct {
	config *Config
}

func NewFilter(config *Config) (*filter, error) {
	return &filter{
		config: config,
	}, nil
}

func (f *filter) IncludeRecord(record *Record) bool {
	return record.Kubernetes.Annotations["xrstf.de/bunker"] != "ignore"
}
