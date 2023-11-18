package server

import (
	"context"

	"github.com/tinkerbell/tink/api/v1alpha2"
	"github.com/tinkerbell/tink/internal/server/index"
	"github.com/tinkerbell/tink/internal/workflow"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type Agent interface {
	RunWorkflow(context.Context, workflow.Workflow) error
	CancelWorkflow(_ context.Context, workflowID string) error
}

type WorkflowDispatcher struct {
	cluster cluster.Cluster
	client  client.WithWatch
}

func (d WorkflowDispatcher) Handle(ctx context.Context, agentID string, agent Agent) error {
	// Find Workflows this agent could run.
	// Dispatch Workflow
	// Watch Workflow for cancellation or end state

	wrkflw, err := d.findInFlightWorkflow(ctx, agentID)
	if err != nil {
		return err
	}

	if wrkflw != nil {
		d.monitor(ctx, wrkflw)
	}

	wrkflw, err := d.findNextWorkflow(ctx, agentID)
	if err != nil {
		return err
	}

	if err := agent.RunWorkflow(ctx, toWorkflow(wrkflw)); err != nil {
		return err
	}

	d.monitor(ctx, wrkflw)
	await(ctx, d.client, func(o client.Object) bool {
		wrkflw := o.(*v1alpha2.Workflow)
		if wrkflow.Status.State
	}, func(o client.Object) {
		d.
	})

	return nil
}

func (d WorkflowDispatcher) awaitWorkflow(id string) error {
	return nil
}

func (d WorkflowDispatcher) findInFlightWorkflow(ctx context.Context, agentID string) (*v1alpha2.Workflow, error) {
	clnt := d.cluster.GetClient()

	// Determine if there are
	var scheduled v1alpha2.WorkflowList
	if err := clnt.List(ctx, &scheduled, client.MatchingFields{
		index.WorkflowState: string(v1alpha2.WorkflowStateScheduled),
	}); err != nil {
		return nil, err
	}

	var running v1alpha2.WorkflowList
	if err := clnt.List(ctx, &running, client.MatchingFields{
		index.WorkflowState: string(v1alpha2.WorkflowStateRunning),
	}); err != nil {
		return nil, err
	}

	return nil, nil
}

func (d WorkflowDispatcher) findNextWorkflow(ctx context.Context, agentID string) (*v1alpha2.Workflow, error) {
	return nil, nil
}

func (d WorkflowDispatcher) monitor(ctx context.Context, wrkflw *v1alpha2.Workflow) error {
	return nil
}

func toWorkflow(*v1alpha2.Workflow) workflow.Workflow {
	return workflow.Workflow{}
}

func await(ctx context.Context, clnt client.WithWatch, cond func(o client.Object) bool, do func(o client.Object)) error {
	return nil
}
