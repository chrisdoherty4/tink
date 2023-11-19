package server

import (
	"context"
	"fmt"

	"github.com/tinkerbell/tink/api/v1alpha2"
	"github.com/tinkerbell/tink/internal/server/index"
	"github.com/tinkerbell/tink/internal/workflow"
	apimwatch "k8s.io/apimachinery/pkg/watch"
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
	wrkflw, err := d.findNextWorkflow(ctx, agentID)
	if err != nil {
		return err
	}

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

func (d WorkflowDispatcher) FindNextWorkflow(ctx context.Context, agentID string) (*v1alpha2.Workflow, error) {
	var wl v1alpha2.WorkflowList
	watch, err := d.client.Watch(ctx, &wl, client.MatchingFields{
		index.WorkflowAgentID: agentID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}

	for {
		select {
		case ev := <-watch.ResultChan():
			fmt.Println(ev)
		}
	}

	return nil, nil
}

func (d WorkflowDispatcher) monitor(ctx context.Context, wrkflw *v1alpha2.Workflow) error {
	return nil
}

func toWorkflow(*v1alpha2.Workflow) workflow.Workflow {
	return workflow.Workflow{}
}

func await(ctx context.Context, clnt client.WithWatch, fn func(o client.Object) error, listOpts ...client.ListOption) error {

	watch, err := clnt.Watch(ctx, &v1alpha2.WorkflowList{}, listOpts...)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-watch.ResultChan():
			switch ev.Type {
			case apimwatch.Added, apimwatch.Deleted, apimwatch.Modified:
				if err := fn(ev.Object); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
