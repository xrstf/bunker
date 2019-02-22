package main

import (
	"regexp"
	"strings"
	"time"
)

type Payload []*Record

type Record struct {
	Date       time.Time          `json:"date"`
	Log        string             `json:"log"`
	Kubernetes KubernetesMetadata `json:"kubernetes"`
}

func (r *Record) StringReplacements() []string {
	t := r.Date.UTC()

	replacements := []string{
		"%year%", t.Format("2006"),
		"%month%", t.Format("01"),
		"%dayofmonth%", t.Format("02"),
		"%date%", t.Format("2006-01-02"),
	}

	return append(replacements, r.Kubernetes.StringReplacements()...)
}

type KubernetesMetadata struct {
	PodName       string            `json:"pod_name"`
	NamespaceName string            `json:"namespace_name"`
	PodID         string            `json:"pod_id"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	Host          string            `json:"host"`
	ContainerName string            `json:"container_name"`
	DockerID      string            `json:"docker_id"`
}

var labelSanitiser = regexp.MustCompile(`[^a-z0-9_]`)
var fsSanitiser = regexp.MustCompile(`[^a-zA-Z0-9_,;-]`)

func (m *KubernetesMetadata) StringReplacements() []string {
	replacements := []string{
		"%kubernetes_pod_name%", m.PodName,
		"%kubernetes_namespace_name%", m.NamespaceName,
		"%kubernetes_pod_id%", m.PodID,
		"%kubernetes_host%", m.Host,
		"%kubernetes_container_name%", m.ContainerName,
		"%kubernetes_docker_id%", m.DockerID,
	}

	for name, value := range m.Labels {
		name = labelSanitiser.ReplaceAllString(strings.ToLower(name), "_")
		value = fsSanitiser.ReplaceAllString(value, "_")

		replacements = append(replacements, name, value)
	}

	return replacements
}
