package deps

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/opendatahub-io/odh-cli/pkg/cmd"
	"github.com/opendatahub-io/odh-cli/pkg/util/iostreams"
)

const graphHeaderWidth = 50

// Verify GraphCommand implements cmd.Command interface at compile time.
var _ cmd.Command = (*GraphCommand)(nil)

// GraphCommand holds the deps graph command configuration.
type GraphCommand struct {
	IO          iostreams.Interface
	ConfigFlags *genericclioptions.ConfigFlags

	Refresh bool
}

// NewGraphCommand creates a new GraphCommand with defaults.
func NewGraphCommand(streams genericiooptions.IOStreams, configFlags *genericclioptions.ConfigFlags) *GraphCommand {
	return &GraphCommand{
		IO:          iostreams.NewIOStreams(streams.In, streams.Out, streams.ErrOut),
		ConfigFlags: configFlags,
	}
}

// AddFlags registers command-specific flags with the provided FlagSet.
func (c *GraphCommand) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Refresh, "refresh", false, "Fetch latest manifest from odh-gitops")
}

// Complete prepares the command for execution.
func (c *GraphCommand) Complete() error {
	return nil
}

// Validate checks the command options.
func (c *GraphCommand) Validate() error {
	return nil
}

// Run executes the graph command.
func (c *GraphCommand) Run(ctx context.Context) error {
	result, err := GetManifest(ctx, c.Refresh)
	if err != nil {
		return fmt.Errorf("get manifest: %w", err)
	}

	manifest := result.Manifest
	w := c.IO.Out()

	_, _ = fmt.Fprintf(w, "Dependency Graph for ODH/RHOAI %s\n", result.Version)
	_, _ = fmt.Fprintln(w, strings.Repeat("=", graphHeaderWidth))
	_, _ = fmt.Fprintln(w)

	deps := manifest.GetDependencies()
	depMap := buildDepMap(manifest)

	for _, dep := range deps {
		c.printDepNode(dep, depMap)
	}

	return nil
}

// buildDepMap creates a map of dependency name to its transitive dependencies.
func buildDepMap(manifest *Manifest) map[string][]string {
	depMap := make(map[string][]string)

	for name, dep := range manifest.Dependencies {
		transDeps := make([]string, 0, len(dep.Dependencies))
		for transName, val := range dep.Dependencies {
			if isEnabled(val) {
				transDeps = append(transDeps, toDisplayName(transName))
			}
		}

		sort.Strings(transDeps)
		depMap[name] = transDeps
	}

	return depMap
}

func (c *GraphCommand) printDepNode(dep DependencyInfo, depMap map[string][]string) {
	w := c.IO.Out()

	_, _ = fmt.Fprintf(w, "%s\n", dep.DisplayName)

	requiredBy := "(none)"
	if len(dep.RequiredBy) > 0 {
		requiredBy = strings.Join(dep.RequiredBy, ", ")
	}

	dependsOn := "(none)"
	if transDeps, ok := depMap[dep.Name]; ok && len(transDeps) > 0 {
		dependsOn = strings.Join(transDeps, ", ")
	}

	_, _ = fmt.Fprintf(w, "├── Required by: %s\n", requiredBy)
	_, _ = fmt.Fprintf(w, "└── Depends on: %s\n", dependsOn)
	_, _ = fmt.Fprintln(w)
}
