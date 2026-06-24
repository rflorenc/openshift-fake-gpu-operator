package resources

import (
	"fmt"
	"os"
)

const (
	defaultImageRegistry = "ghcr.io/run-ai/fake-gpu-operator"
	defaultImageTag      = "0.1.0"
)

var componentRepos = map[string]string{
	"device-plugin":             "device-plugin",
	"status-updater":            "status-updater",
	"metrics-exporter":          "status-exporter",
	"topology-server":           "topology-server",
	"mig-faker":                 "mig-faker",
	"dra-plugin":                "dra-plugin-gpu",
	"compute-domain-controller": "compute-domain-controller",
	"compute-domain-dra":        "compute-domain-dra-plugin",
}

func DefaultRegistry() string {
	if v := os.Getenv("RELATED_IMAGE_REGISTRY"); v != "" {
		return v
	}
	return defaultImageRegistry
}

func DefaultTag() string {
	if v := os.Getenv("RELATED_IMAGE_TAG"); v != "" {
		return v
	}
	return defaultImageTag
}

func Image(component, registry, tag string, overrides map[string]string) string {
	if img, ok := overrides[component]; ok {
		return img
	}
	if registry == "" {
		registry = DefaultRegistry()
	}
	if tag == "" {
		tag = DefaultTag()
	}
	return fmt.Sprintf("%s/%s:%s", registry, componentRepos[component], tag)
}
