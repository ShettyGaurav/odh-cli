package client

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opendatahub-io/odh-cli/pkg/resources"
	clierrors "github.com/opendatahub-io/odh-cli/pkg/util/errors"
)

// Well-known operator namespace defaults.
const (
	DefaultRHOAIOperatorNamespace = "redhat-ods-operator"
	DefaultODHOperatorNamespace   = "opendatahub"
	DefaultOpenShiftOperatorsNS   = "openshift-operators"
)

// OperatorInfo holds operator namespace and deployment name discovered from OLM.
type OperatorInfo struct {
	Namespace      string
	DeploymentName string
}

// DiscoverOperatorFromOLM searches for the operator CSV across all namespaces.
// Finds CSVs by name prefix (rhods-operator or opendatahub-operator) and returns
// both the namespace and deployment name from a single API call.
// Returns nil if OLM is not available or no matching CSV is found.
func DiscoverOperatorFromOLM(ctx context.Context, c Reader) *OperatorInfo {
	if !c.OLM().Available() {
		return nil
	}

	csvList, err := c.OLM().ClusterServiceVersions("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	for _, csv := range csvList.Items {
		name := csv.GetName()
		if strings.HasPrefix(name, "rhods-operator.") || strings.HasPrefix(name, "opendatahub-operator.") {
			info := &OperatorInfo{}

			// Get namespace - use original namespace from olm.copiedFrom if this is a copy
			if copiedFrom, ok := csv.GetLabels()["olm.copiedFrom"]; ok && copiedFrom != "" {
				info.Namespace = copiedFrom
			} else {
				info.Namespace = csv.GetNamespace()
			}

			// Get deployment name from install strategy
			deployments := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
			if len(deployments) > 0 {
				info.DeploymentName = deployments[0].Name
			}

			return info
		}
	}

	return nil
}

// DiscoverOperatorNamespace finds the operator namespace using well-known defaults
// with fallback to OLM discovery. Returns an error if no operator is found.
func DiscoverOperatorNamespace(ctx context.Context, c Reader) (string, error) {
	// Try well-known defaults first (fast - max 6 API calls)
	ns, err := discoverOperatorNamespaceFromDefaults(ctx, c)
	if err == nil {
		return ns, nil
	}

	// Fall back to OLM-based discovery (slower but comprehensive)
	if info := DiscoverOperatorFromOLM(ctx, c); info != nil && info.Namespace != "" {
		return info.Namespace, nil
	}

	return "", clierrors.ErrOperatorNamespaceNotFound()
}

// DiscoverOperatorNamespaceWithInfo finds the operator namespace using pre-fetched
// OLM info, falling back to well-known defaults if info is nil or incomplete.
func DiscoverOperatorNamespaceWithInfo(ctx context.Context, c Reader, info *OperatorInfo) (string, error) {
	if info != nil && info.Namespace != "" {
		return info.Namespace, nil
	}

	return discoverOperatorNamespaceFromDefaults(ctx, c)
}

// discoverOperatorNamespaceFromDefaults tries well-known operator namespaces
// by checking if the specific operator deployment exists there.
// Returns the first namespace where the operator is found, or an error.
// NotFound errors are silently skipped; other errors (Forbidden, timeout) are returned.
func discoverOperatorNamespaceFromDefaults(ctx context.Context, c Reader) (string, error) {
	operatorDeploymentNames := []string{
		"rhods-operator",
		"opendatahub-operator-controller-manager",
	}

	namespaces := []string{
		DefaultRHOAIOperatorNamespace,
		DefaultODHOperatorNamespace,
		DefaultOpenShiftOperatorsNS,
	}

	var firstNonNotFoundErr error

	for _, ns := range namespaces {
		for _, name := range operatorDeploymentNames {
			_, err := c.GetResource(ctx, resources.Deployment, name, InNamespace(ns))
			if err == nil {
				return ns, nil
			}

			// Only skip NotFound errors; capture other errors (Forbidden, timeout, etc.)
			if !apierrors.IsNotFound(err) && firstNonNotFoundErr == nil {
				if classified := clierrors.Classify(err); classified != nil {
					firstNonNotFoundErr = classified
				} else {
					firstNonNotFoundErr = err
				}
			}
		}
	}

	// If we encountered a non-NotFound error, return it
	if firstNonNotFoundErr != nil {
		return "", firstNonNotFoundErr
	}

	return "", clierrors.ErrOperatorNamespaceNotFound()
}
