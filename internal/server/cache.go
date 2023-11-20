package server

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/tinkerbell/tink/api/v1alpha2"
	"github.com/tinkerbell/tink/internal/workflow"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimwatch "k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkflowProvider struct {
	mu        sync.Mutex
	workflows map[string][]*v1alpha2.Workflow
	observers map[string]<-chan workflow.Workflow
	client    client.WithWatch
	log       logr.Logger
}

func (wp *WorkflowProvider) Start(ctx context.Context) error {
	var wl v1alpha2.WorkflowList
	watch, err := wp.client.Watch(ctx, &wl)
	if err != nil {
		return err
	}
	defer watch.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-watch.ResultChan():
			if !ok {
				return errors.New("watch channel unexpectedly closed")
			}
			// Convert runtime.Object to client.Object (controller runtime)
			wf, ok := ev.Object.(*v1alpha2.Workflow)
			if !ok {
				// TODO(chrisdoherty4) Handle error, probably a log event.
				_ = ok
			}

			wp.mu.Lock()
			switch ev.Type {
			case apimwatch.Added:
				agentID := wf.Status.AgentID

				if _, exists := wp.workflows[agentID]; !exists {
					wp.workflows[agentID] = make([]*v1alpha2.Workflow, 0, 1)
				}

				wp.workflows[agentID] = append(wp.workflows[agentID], wf)
			case apimwatch.Modified:
				wp.log.Info("Received modified object", "obj", ev.Object.(metav1.Object).GetName())
			}
			wp.mu.Unlock()
		}
	}
}

func (wp *WorkflowProvider) GetNext(ctx context.Context, agentID string) (workflow.Workflow, WorkflowMonitor, error) {
	// Lock mu
	// Check for workflows for the agentID
	// If none register an observer
	// Unlock mu
	// Listen for workflows.

	wp.mu.Lock()

	// TODO(chrisdoherty4) Ensure the mutex also protects the arrays in the cache. We can probably optimize
	// to unlock the map and protect against the queue only freeing up the cache for other threads.
	q, exists := wp.workflows[agentID]
	if exists && len(q) > 0 {
		wf := q[0]
		q = q[1:]
		wp.workflows[agentID] = q
		wp.mu.Unlock()
		return toWorkflow(wf), nil, nil
	}

	// There aren't any Workflows so we need to register an observer.
	recv := make(chan workflow.Workflow)
	wp.observers[agentID] = recv

	wp.mu.Unlock()

	// TODO(chrisdoherty4) Separate Workflow mutex from Observer mutex because we don't need to hold up the
	// workflow mutex when we unregister the observer.
	defer func() {
		wp.mu.Lock()
		defer wp.mu.Unlock()
		delete(wp.observers, agentID)
	}()

	select {
	case <-ctx.Done():
		return workflow.Workflow{}, nil, ctx.Err()
	case wf := <-recv:
		return wf, nil, nil
	}
}
