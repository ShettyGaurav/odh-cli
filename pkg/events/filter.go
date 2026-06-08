package events

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/opendatahub-io/opendatahub-operator/pkg/clusterhealth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opendatahub-io/odh-cli/pkg/resources"
	clierrors "github.com/opendatahub-io/odh-cli/pkg/util/errors"
)

const eventFetchTimeout = 30 * time.Second

// fetchEvents retrieves events using the clusterhealth library.
func (c *Command) fetchEvents(ctx context.Context) ([]clusterhealth.EventInfo, error) {
	namespaces, err := c.getTargetNamespaces()
	if err != nil {
		return nil, err
	}

	fetchCtx, cancel := context.WithTimeout(ctx, eventFetchTimeout)
	defer cancel()

	cfg := clusterhealth.RecentEventsConfig{
		Client:     c.crClient,
		Namespaces: namespaces,
		Since:      c.Since,
		EventType:  c.EventType,
	}

	events, err := clusterhealth.RunRecentEvents(fetchCtx, cfg)
	if err != nil {
		return nil, clierrors.ErrEventsFetchFailed(err)
	}

	return events, nil
}

// getTargetNamespaces returns the namespaces to query for events.
// For -n <namespace>, returns ONLY that namespace (exclusive scope like kubectl).
// For --all-namespaces or no flags, returns ODH namespaces (apps, operator, monitoring).
// Returns ErrNoNamespacesDiscovered if no namespaces could be determined.
func (c *Command) getTargetNamespaces() ([]string, error) {
	// If user explicitly passed -n <namespace>, return ONLY that namespace (exclusive)
	// Note: NamespaceExplicit distinguishes "odh events -n foo" from "odh events" (no flags)
	if c.NamespaceExplicit && c.Namespace != "" {
		return []string{c.Namespace}, nil
	}

	// Otherwise return ODH namespaces (for -A or no flags)
	seen := make(map[string]bool)
	var namespaces []string

	add := func(ns string) {
		if ns != "" && !seen[ns] {
			seen[ns] = true
			namespaces = append(namespaces, ns)
		}
	}

	add(c.ApplicationsNS)
	add(c.OperatorNS)
	add(c.MonitoringNS)

	if len(namespaces) == 0 {
		return nil, clierrors.ErrNoNamespacesDiscovered()
	}

	return namespaces, nil
}

// sortEventsByTime sorts events in place by timestamp, most recent first.
// Uses stable sort to preserve original order for events with identical timestamps.
func sortEventsByTime(events []clusterhealth.EventInfo) {
	slices.SortStableFunc(events, func(a, b clusterhealth.EventInfo) int {
		return cmp.Compare(b.LastTime.UnixNano(), a.LastTime.UnixNano())
	})
}

// filterEventsByComponent filters events to only those related to a component.
// It looks up each event's InvolvedObject and checks for the component label.
func (c *Command) filterEventsByComponent(ctx context.Context, events []clusterhealth.EventInfo) []clusterhealth.EventInfo {
	targetLabel := resources.GetComponentLabelValue(c.Component)
	labelCache := make(map[string]bool)

	var filtered []clusterhealth.EventInfo

	for _, event := range events {
		cacheKey := fmt.Sprintf("%s/%s/%s/%s", targetLabel, event.Namespace, event.Kind, event.Name)

		hasLabel, found := labelCache[cacheKey]
		if !found {
			hasLabel = c.checkObjectHasComponentLabel(ctx, event.Namespace, event.Kind, event.Name, targetLabel)
			labelCache[cacheKey] = hasLabel
		}

		if hasLabel {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// checkObjectHasComponentLabel checks if an object has the component label.
func (c *Command) checkObjectHasComponentLabel(ctx context.Context, namespace, kind, name, labelValue string) bool {
	gvr := kindToGVR(kind)
	if gvr.Resource == "" {
		return false
	}

	var obj interface {
		GetLabels() map[string]string
	}

	if namespace != "" {
		unstr, err := c.Client.Dynamic().Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false
		}

		obj = unstr
	} else {
		unstr, err := c.Client.Dynamic().Resource(gvr).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false
		}

		obj = unstr
	}

	labels := obj.GetLabels()

	return labels[resources.ComponentLabelKey] == labelValue
}

// kindToGVR maps Kubernetes Kind to GroupVersionResource.
func kindToGVR(kind string) schema.GroupVersionResource {
	switch kind {
	case "Pod":
		return resources.Pod.GVR()
	case "Deployment":
		return resources.Deployment.GVR()
	case "ReplicaSet":
		return resources.ReplicaSet.GVR()
	case "StatefulSet":
		return resources.StatefulSet.GVR()
	case "DaemonSet":
		return resources.DaemonSet.GVR()
	case "Service":
		return resources.Service.GVR()
	case "ConfigMap":
		return resources.ConfigMap.GVR()
	case "Secret":
		return resources.Secret.GVR()
	case "Job":
		return resources.Job.GVR()
	default:
		return schema.GroupVersionResource{}
	}
}
