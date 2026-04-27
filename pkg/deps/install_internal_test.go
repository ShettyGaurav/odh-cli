package deps

import (
	"bytes"
	"context"
	"testing"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/opendatahub-io/odh-cli/pkg/util/iostreams"

	. "github.com/onsi/gomega"
)

func TestShouldInstallDep(t *testing.T) {
	tests := []struct {
		name            string
		targetDep       string
		includeOptional bool
		dep             DependencyInfo
		want            bool
	}{
		{
			name:      "target dep matches - always install",
			targetDep: "cert-manager",
			dep:       DependencyInfo{Name: "cert-manager", Enabled: "false"},
			want:      true,
		},
		{
			name:      "target dep matches optional - always install",
			targetDep: "kueue",
			dep:       DependencyInfo{Name: "kueue", Enabled: "auto"},
			want:      true,
		},
		{
			name:      "required dep - install",
			targetDep: "",
			dep:       DependencyInfo{Name: "cert-manager", Enabled: "true"},
			want:      true,
		},
		{
			name:      "optional dep without flag - skip",
			targetDep: "",
			dep:       DependencyInfo{Name: "kueue", Enabled: "auto"},
			want:      false,
		},
		{
			name:      "disabled dep without flag - skip",
			targetDep: "",
			dep:       DependencyInfo{Name: "servicemesh", Enabled: "false"},
			want:      false,
		},
		{
			name:            "optional dep with flag - install",
			targetDep:       "",
			includeOptional: true,
			dep:             DependencyInfo{Name: "kueue", Enabled: "auto"},
			want:            true,
		},
		{
			name:            "disabled dep with flag - still skip",
			targetDep:       "",
			includeOptional: true,
			dep:             DependencyInfo{Name: "servicemesh", Enabled: "false"},
			want:            false,
		},
		{
			name:      "different target dep - use normal rules",
			targetDep: "other-dep",
			dep:       DependencyInfo{Name: "kueue", Enabled: "auto"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			streams := genericiooptions.IOStreams{
				Out:    &bytes.Buffer{},
				ErrOut: &bytes.Buffer{},
			}

			cmd := &InstallCommand{
				IO:              iostreams.NewIOStreams(streams.In, streams.Out, streams.ErrOut),
				TargetDep:       tt.targetDep,
				IncludeOptional: tt.includeOptional,
			}

			got := cmd.shouldInstallDep(tt.dep)
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func TestRunDryRun(t *testing.T) {
	t.Run("with deps to install", func(t *testing.T) {
		g := NewWithT(t)

		var buf bytes.Buffer
		streams := genericiooptions.IOStreams{
			Out:    &buf,
			ErrOut: &bytes.Buffer{},
		}

		cmd := &InstallCommand{
			IO: iostreams.NewIOStreams(streams.In, streams.Out, streams.ErrOut),
		}

		deps := []DependencyInfo{
			{
				Name:         "cert-manager",
				DisplayName:  "Cert Manager",
				Namespace:    "cert-manager-operator",
				Subscription: "cert-manager",
				Channel:      "stable",
				Enabled:      "true",
			},
		}

		err := cmd.runDryRun(context.Background(), deps)

		g.Expect(err).ToNot(HaveOccurred())

		output := buf.String()
		g.Expect(output).To(ContainSubstring("[DRY RUN]"))
		g.Expect(output).To(ContainSubstring("Cert Manager"))
		g.Expect(output).To(ContainSubstring("kind: Namespace"))
		g.Expect(output).To(ContainSubstring("kind: OperatorGroup"))
		g.Expect(output).To(ContainSubstring("kind: Subscription"))
		g.Expect(output).To(ContainSubstring("cert-manager-operator"))
		g.Expect(output).To(ContainSubstring("app.kubernetes.io/managed-by: odh-cli"))
	})

	t.Run("no deps to install", func(t *testing.T) {
		g := NewWithT(t)

		var buf bytes.Buffer
		streams := genericiooptions.IOStreams{
			Out:    &buf,
			ErrOut: &bytes.Buffer{},
		}

		cmd := &InstallCommand{
			IO: iostreams.NewIOStreams(streams.In, streams.Out, streams.ErrOut),
		}

		// Optional dep without --include-optional flag
		deps := []DependencyInfo{
			{
				Name:    "kueue",
				Enabled: "auto",
			},
		}

		err := cmd.runDryRun(context.Background(), deps)

		g.Expect(err).ToNot(HaveOccurred())

		output := buf.String()
		g.Expect(output).To(ContainSubstring("[DRY RUN]"))
		g.Expect(output).To(ContainSubstring("No dependencies to install"))
	})

	t.Run("with target dep", func(t *testing.T) {
		g := NewWithT(t)

		var buf bytes.Buffer
		streams := genericiooptions.IOStreams{
			Out:    &buf,
			ErrOut: &bytes.Buffer{},
		}

		cmd := &InstallCommand{
			IO:        iostreams.NewIOStreams(streams.In, streams.Out, streams.ErrOut),
			TargetDep: "kueue",
		}

		deps := []DependencyInfo{
			{
				Name:        "cert-manager",
				DisplayName: "Cert Manager",
				Namespace:   "cert-manager-operator",
				Enabled:     "true",
			},
			{
				Name:         "kueue",
				DisplayName:  "Kueue",
				Namespace:    "openshift-kueue-operator",
				Subscription: "kueue-operator",
				Channel:      "stable",
				Enabled:      "auto",
			},
		}

		err := cmd.runDryRun(context.Background(), deps)

		g.Expect(err).ToNot(HaveOccurred())

		output := buf.String()
		// Should only show kueue, not cert-manager
		g.Expect(output).To(ContainSubstring("Kueue"))
		g.Expect(output).ToNot(ContainSubstring("Cert Manager"))
	})
}
