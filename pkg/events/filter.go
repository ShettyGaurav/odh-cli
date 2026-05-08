package events

import (
	"cmp"
	"context"
	"slices"

	"github.com/opendatahub-io/opendatahub-operator/pkg/clusterhealth"

	clierrors "github.com/opendatahub-io/odh-cli/pkg/util/errors"
)

// fetchEvents retrieves events using the clusterhealth library.
func (c *Command) fetchEvents(ctx context.Context) ([]clusterhealth.EventInfo, error) {
	namespaces, err := c.getTargetNamespaces()
	if err != nil {
		return nil, err
	}

	cfg := clusterhealth.RecentEventsConfig{
		Client:     c.crClient,
		Namespaces: namespaces,
		Since:      c.Since,
		EventType:  c.EventType,
	}

	events, err := clusterhealth.RunRecentEvents(ctx, cfg)
	if err != nil {
		return nil, clierrors.ErrEventsFetchFailed(err)
	}

	return events, nil
}

// getTargetNamespaces returns the namespaces to query for events.
// For --all-namespaces, returns ODH namespaces (apps, operator, monitoring).
// Otherwise includes the current namespace context as well.
// Returns ErrNoNamespacesDiscovered if no namespaces could be determined.
func (c *Command) getTargetNamespaces() ([]string, error) {
	seen := make(map[string]bool)
	var namespaces []string

	add := func(ns string) {
		if ns != "" && !seen[ns] {
			seen[ns] = true
			namespaces = append(namespaces, ns)
		}
	}

	// Always include ODH namespaces
	add(c.ApplicationsNS)
	add(c.OperatorNS)
	add(c.MonitoringNS)

	// Include current namespace context unless --all-namespaces
	if !c.AllNamespaces && c.Namespace != "" {
		add(c.Namespace)
	}

	if len(namespaces) == 0 {
		return nil, clierrors.ErrNoNamespacesDiscovered()
	}

	return namespaces, nil
}

// sortEventsByTime sorts events in place by timestamp, most recent first.
func sortEventsByTime(events []clusterhealth.EventInfo) {
	slices.SortFunc(events, func(a, b clusterhealth.EventInfo) int {
		return cmp.Compare(b.LastTime.UnixNano(), a.LastTime.UnixNano())
	})
}
