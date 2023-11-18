package index

import (
	"github.com/tinkerbell/tink/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkflowState indexes a Workflow object on the .Status.State field.
var WorkflowState = ".Status.State"

// WorkflowStateUnderway is a hybrid state indicating a Workflow is either Scheduled or Running.
// It is useful for listening Workflows efficiently using the WorkflowState index.
var WorkflowStateUnderway = "Underway"

// WorkflowStateFn is the indexing function for WorkflowState.
func WorkflowStateFn(o client.Object) []string {
	w := o.(*v1alpha2.Workflow)

	state := w.Status.State
	states := []string{string(state)}

	if state == v1alpha2.WorkflowStateScheduled || state == v1alpha2.WorkflowStateRunning {
		states = append(states, WorkflowStateUnderway)
	}

	return states
}
