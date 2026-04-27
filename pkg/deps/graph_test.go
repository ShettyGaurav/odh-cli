package deps_test

import (
	"bytes"
	"testing"

	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/opendatahub-io/odh-cli/pkg/deps"

	. "github.com/onsi/gomega"
)

const (
	testGraphHeader     = "Dependency Graph"
	testGraphCertMgr    = "Cert Manager"
	testGraphRequiredBy = "Required by:"
	testGraphDependsOn  = "Depends on:"
)

func TestGraphCommand(t *testing.T) {
	t.Run("Complete", func(t *testing.T) {
		g := NewWithT(t)

		streams := genericiooptions.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		}

		cmd := deps.NewGraphCommand(streams, nil)

		err := cmd.Complete()

		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("Validate", func(t *testing.T) {
		g := NewWithT(t)

		streams := genericiooptions.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		}

		cmd := deps.NewGraphCommand(streams, nil)

		err := cmd.Validate()

		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("AddFlags", func(t *testing.T) {
		g := NewWithT(t)

		streams := genericiooptions.IOStreams{
			Out:    &bytes.Buffer{},
			ErrOut: &bytes.Buffer{},
		}

		cmd := deps.NewGraphCommand(streams, nil)

		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		cmd.AddFlags(fs)

		g.Expect(fs.Lookup("refresh")).ToNot(BeNil())
	})

	t.Run("Run", func(t *testing.T) {
		g := NewWithT(t)

		out := &bytes.Buffer{}
		streams := genericiooptions.IOStreams{
			Out:    out,
			ErrOut: &bytes.Buffer{},
		}

		cmd := deps.NewGraphCommand(streams, nil)

		err := cmd.Run(t.Context())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(out.String()).To(ContainSubstring(testGraphHeader))
		g.Expect(out.String()).To(ContainSubstring(testGraphCertMgr))
		g.Expect(out.String()).To(ContainSubstring(testGraphRequiredBy))
		g.Expect(out.String()).To(ContainSubstring(testGraphDependsOn))
	})
}
