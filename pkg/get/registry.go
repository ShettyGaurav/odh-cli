package get

import (
	"fmt"
	"sort"
	"strings"

	"github.com/opendatahub-io/odh-cli/pkg/resources"
)

// resourceEntry maps a canonical resource name to its ResourceType.
//
//nolint:gochecknoglobals // Registry of supported get-command resource types
var resourceMap = map[string]resources.ResourceType{
	"notebooks":                        resources.Notebook,
	"inferenceservices":                resources.InferenceService,
	"servingruntimes":                  resources.ServingRuntime,
	"datasciencepipelinesapplications": resources.DataSciencePipelinesApplicationV1,
}

// aliasMap maps short aliases to canonical resource names.
//
//nolint:gochecknoglobals // Registry of short aliases for resource types
var aliasMap = map[string]string{
	"nb":       "notebooks",
	"isvc":     "inferenceservices",
	"sr":       "servingruntimes",
	"pipeline": "datasciencepipelinesapplications",
}

// Resolve looks up a resource name or alias and returns the corresponding ResourceType.
// It checks aliases first, then canonical names.
func Resolve(name string) (resources.ResourceType, error) {
	lower := strings.ToLower(name)

	if canonical, ok := aliasMap[lower]; ok {
		return resourceMap[canonical], nil
	}

	if rt, ok := resourceMap[lower]; ok {
		return rt, nil
	}

	return resources.ResourceType{}, fmt.Errorf(
		"unknown resource type %q (available: %s)",
		name, strings.Join(Names(), ", "),
	)
}

// Names returns a sorted list of all valid resource names and aliases
// for use in help text and tab completion.
func Names() []string {
	seen := make(map[string]struct{}, len(resourceMap)+len(aliasMap))

	for name := range resourceMap {
		seen[name] = struct{}{}
	}

	for alias := range aliasMap {
		seen[alias] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
