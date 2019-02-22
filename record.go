package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Payload struct {
	Tag     string
	Records []*Record
}

type Record struct {
	Date       time.Time          `json:"date"`
	Log        string             `json:"log"`
	Kubernetes KubernetesMetadata `json:"kubernetes"`
}

func (r *Record) StringReplacements(tag string) []string {
	t := r.Date.UTC()

	replacements := make([]string, 0)
	replacements = addReplacement(replacements, "year", t.Format("2006"))
	replacements = addReplacement(replacements, "month", t.Format("01"))
	replacements = addReplacement(replacements, "dayofmonth", t.Format("02"))
	replacements = addReplacement(replacements, "date", t.Format("2006-01-02"))
	replacements = addReplacement(replacements, "tag", tag)

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

func (m *KubernetesMetadata) StringReplacements() []string {
	replacements := make([]string, 0)
	replacements = addReplacement(replacements, "kubernetes_pod_name", m.PodName)
	replacements = addReplacement(replacements, "kubernetes_namespace_name", m.NamespaceName)
	replacements = addReplacement(replacements, "kubernetes_pod_id", m.PodID)
	replacements = addReplacement(replacements, "kubernetes_host", m.Host)
	replacements = addReplacement(replacements, "kubernetes_container_name", m.ContainerName)
	replacements = addReplacement(replacements, "kubernetes_docker_id", m.DockerID)

	for name, value := range m.Labels {
		name = fmt.Sprintf("kubernetes_label_%s", labelSanitiser.ReplaceAllString(strings.ToLower(name), "_"))

		replacements = addReplacement(replacements, name, value)
	}

	return replacements
}

var fsSanitiser = regexp.MustCompile(`[^a-zA-Z0-9_,;. -]`)

func addReplacement(list []string, name string, value string) []string {
	if value == "" {
		value = fmt.Sprintf("NO_%s", strings.ToUpper(name))
	}

	name = fmt.Sprintf("%%%s%%", name)
	list = append(list, name, fsSanitiser.ReplaceAllString(value, "_"))

	return list
}
