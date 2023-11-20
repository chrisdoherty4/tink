package server

import (
	"context"

	"github.com/tinkerbell/tink/api/v1alpha2"
	"github.com/tinkerbell/tink/internal/workflow"
)

type Agent interface {
	RunWorkflow(context.Context, workflow.Workflow) error
	CancelWorkflow(_ context.Context, workflowID string) error
}

type WorkflowMonitor interface {
	Deleted() <-chan struct{}
	Completed() <-chan struct{}
}

type WorkflowDispatcher struct {
	provider WorkflowProvider
}

func (d WorkflowDispatcher) Handle(ctx context.Context, agentID string, agent Agent) error {
	for {
		// If the context is done we can exit early.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wf, monitor, err := d.provider.GetNext(ctx, agentID)
		if err != nil {
			return err
		}

		if err := agent.RunWorkflow(ctx, wf); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-monitor.Deleted():
			if err := agent.CancelWorkflow(ctx, wf.ID); err != nil {
				return err
			}
		case <-monitor.Completed():
		}
	}
}

func toWorkflow(*v1alpha2.Workflow) workflow.Workflow {
	return workflow.Workflow{}
}
