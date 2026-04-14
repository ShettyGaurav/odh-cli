//nolint:testpackage // internal test: exercises unexported evaluateVerdict method
package lint

import (
	"bytes"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/opendatahub-io/odh-cli/pkg/lint/check"
	"github.com/opendatahub-io/odh-cli/pkg/lint/check/result"
	clierrors "github.com/opendatahub-io/odh-cli/pkg/util/errors"

	. "github.com/onsi/gomega"
)

const (
	testVerdictGroup       = "test"
	testVerdictKind        = "resource"
	testVerdictCheckName   = "check"
	testVerdictDescription = "test check"
)

func newTestCommand() *Command {
	var out, errOut bytes.Buffer
	streams := genericiooptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &out,
		ErrOut: &errOut,
	}

	return NewCommand(streams, genericclioptions.NewConfigFlags(true))
}

func buildExecution(impact result.Impact) check.CheckExecution {
	dr := result.New(testVerdictGroup, testVerdictKind, testVerdictCheckName, testVerdictDescription)
	dr.SetCondition(check.NewCondition(
		check.ConditionTypeAvailable,
		metav1.ConditionFalse,
		check.WithReason(check.ReasonResourceNotFound),
		check.WithMessage("test finding"),
		check.WithImpact(impact),
	))

	return check.CheckExecution{Result: dr}
}

func buildPassingExecution() check.CheckExecution {
	dr := result.New(testVerdictGroup, testVerdictKind, testVerdictCheckName, testVerdictDescription)
	dr.SetCondition(check.NewCondition(
		check.ConditionTypeAvailable,
		metav1.ConditionTrue,
		check.WithReason(check.ReasonResourceAvailable),
		check.WithMessage("all good"),
	))

	return check.CheckExecution{Result: dr}
}

func TestEvaluateVerdict(t *testing.T) {
	cases := []struct {
		name              string
		results           []check.CheckExecution
		wantErr           bool
		wantCode          clierrors.ExitCode
		notAlreadyHandled bool
	}{
		{
			name:    "should return nil for all-passing results",
			results: []check.CheckExecution{buildPassingExecution()},
		},
		{
			name:     "should return ExitError for prohibited findings",
			results:  []check.CheckExecution{buildExecution(result.ImpactProhibited)},
			wantErr:  true,
			wantCode: clierrors.ExitError,
		},
		{
			name:     "should return ExitError for blocking findings",
			results:  []check.CheckExecution{buildExecution(result.ImpactBlocking)},
			wantErr:  true,
			wantCode: clierrors.ExitError,
		},
		{
			name:     "should return ExitWarning for advisory-only findings",
			results:  []check.CheckExecution{buildExecution(result.ImpactAdvisory)},
			wantErr:  true,
			wantCode: clierrors.ExitWarning,
		},
		{
			name: "should return ExitError when both prohibited and advisory findings exist",
			results: []check.CheckExecution{
				buildExecution(result.ImpactProhibited),
				buildExecution(result.ImpactAdvisory),
			},
			wantErr:           true,
			wantCode:          clierrors.ExitError,
			notAlreadyHandled: true,
		},
		{
			name: "should return ExitError when both blocking and advisory findings exist",
			results: []check.CheckExecution{
				buildExecution(result.ImpactBlocking),
				buildExecution(result.ImpactAdvisory),
			},
			wantErr:  true,
			wantCode: clierrors.ExitError,
		},
		{
			name: "should skip nil results",
			results: []check.CheckExecution{
				{Result: nil},
				buildExecution(result.ImpactAdvisory),
			},
			wantErr:  true,
			wantCode: clierrors.ExitWarning,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			cmd := newTestCommand()
			cmd.OutputFormat = OutputFormatJSON

			err := cmd.evaluateVerdict(tc.results)
			if tc.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(clierrors.ExitCodeFromError(err)).To(Equal(tc.wantCode))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tc.notAlreadyHandled {
				g.Expect(errors.Is(err, clierrors.ErrAlreadyHandled)).To(BeFalse())
			}
		})
	}
}
