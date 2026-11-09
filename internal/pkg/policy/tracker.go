package policy

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// maxListedConflicts is the maximum amount of conflicted workloads listed in the Accepted condition message.
const maxListedConflicts = 10

// Tracker collects, during a scan, which workloads every policy was applied to and which ones it lost.
// It is safe for concurrent use.
type Tracker struct {
	mutex   sync.Mutex
	entries map[string]*trackerEntry
}

type trackerEntry struct {
	policy     *Policy
	matched    int32
	conflicted []string
}

// NewTracker creates a tracker for the given policies.
func NewTracker(policies []*Policy) *Tracker {
	tracker := &Tracker{entries: make(map[string]*trackerEntry, len(policies))}

	for _, policy := range policies {
		tracker.entries[policy.Key()] = &trackerEntry{policy: policy}
	}

	return tracker
}

// Record records that winner was applied to the workload and that losers also matched it but were not applied.
func (t *Tracker) Record(winner *Policy, losers []*Policy, workload string) {
	if t == nil || winner == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if entry, ok := t.entries[winner.Key()]; ok {
		entry.matched++
	}

	for _, loser := range losers {
		entry, ok := t.entries[loser.Key()]
		if !ok {
			continue
		}

		entry.conflicted = append(entry.conflicted, fmt.Sprintf("%s (applied: %s)", workload, winner.Object.Name))
	}
}

// StatusUpdate is the status a policy should have after a scan.
type StatusUpdate struct {
	Policy *Policy
	Status v1alpha1.DownscalePolicyStatus
}

// PendingUpdates returns the status updates of all policies whose status changed during the scan.
func (t *Tracker) PendingUpdates() []StatusUpdate {
	if t == nil {
		return nil
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	keys := make([]string, 0, len(t.entries))
	for key := range t.entries {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	updates := make([]StatusUpdate, 0, len(keys))

	for _, key := range keys {
		entry := t.entries[key]

		status := entry.desiredStatus()
		if equality.Semantic.DeepEqual(status, entry.policy.Object.Status) {
			continue
		}

		updates = append(updates, StatusUpdate{Policy: entry.policy, Status: status})
	}

	return updates
}

// desiredStatus computes the status of the policy from the result of the scan.
func (e *trackerEntry) desiredStatus() v1alpha1.DownscalePolicyStatus {
	object := e.policy.Object
	status := *object.Status.DeepCopy()
	status.ObservedGeneration = object.Generation

	if !e.policy.Valid() {
		status.MatchedWorkloads = 0
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               v1alpha1.ConditionTypeValid,
			Status:             metav1.ConditionFalse,
			Reason:             v1alpha1.ReasonInvalidValue,
			Message:            e.policy.Err.Error(),
			ObservedGeneration: object.Generation,
		})
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               v1alpha1.ConditionTypeAccepted,
			Status:             metav1.ConditionFalse,
			Reason:             v1alpha1.ReasonInvalid,
			Message:            "the policy is not applied because it is not valid",
			ObservedGeneration: object.Generation,
		})

		return status
	}

	status.MatchedWorkloads = e.matched
	meta.SetStatusCondition(&status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionTypeValid,
		Status:             metav1.ConditionTrue,
		Reason:             v1alpha1.ReasonParsed,
		Message:            "all scaling values were parsed successfully",
		ObservedGeneration: object.Generation,
	})

	if len(e.conflicted) == 0 {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               v1alpha1.ConditionTypeAccepted,
			Status:             metav1.ConditionTrue,
			Reason:             v1alpha1.ReasonAccepted,
			Message:            "the policy is applied to all workloads it matches",
			ObservedGeneration: object.Generation,
		})

		return status
	}

	meta.SetStatusCondition(&status.Conditions, metav1.Condition{
		Type:               v1alpha1.ConditionTypeAccepted,
		Status:             metav1.ConditionFalse,
		Reason:             v1alpha1.ReasonConflicted,
		Message:            conflictMessage(e.conflicted),
		ObservedGeneration: object.Generation,
	})

	return status
}

func conflictMessage(conflicted []string) string {
	sorted := slices.Clone(conflicted)
	slices.Sort(sorted)

	listed := sorted
	if len(listed) > maxListedConflicts {
		listed = listed[:maxListedConflicts]
	}

	message := fmt.Sprintf(
		"another policy takes precedence on %d matched workload(s): %s",
		len(sorted),
		strings.Join(listed, ", "),
	)

	if remaining := len(sorted) - len(listed); remaining > 0 {
		message += fmt.Sprintf(" and %d more", remaining)
	}

	return message
}
