package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/tink/internal/agent/event"
	"github.com/tinkerbell/tink/internal/agent/workflow"
	workflowproto "github.com/tinkerbell/tink/internal/proto/workflow/v2"
)

var _ event.Recorder = &GRPC{}

type GRPC struct {
	log    logr.Logger
	client workflowproto.WorkflowServiceClient
}

func NewGRPC(log logr.Logger, client workflowproto.WorkflowServiceClient) *GRPC {
	return &GRPC{
		log:    log,
		client: client,
	}
}

func (g *GRPC) Start(ctx context.Context, agentID string, handler WorkflowHandler) error {
	stream, err := g.client.StreamWorkflows(ctx, &workflowproto.StreamWorkflowsRequest{
		AgentId: agentID,
	})
	if err != nil {
		return err
	}

	log := g.log
	var idx workflowIndex

	for {
		request, err := stream.Recv()
		switch {
		case errors.Is(err, io.EOF):
			// TODO(chrisdoherty4) Think about cancelling
			return nil
		case err != nil:
			return err
		}

		switch request.GetCmd().(type) {
		case *workflowproto.StreamWorkflowsResponse_StartWorkflow_:
			grpcWorkflow := request.GetStartWorkflow().GetWorkflow()

			if err := validateGRPCWorkflow(grpcWorkflow); err != nil {
				log.Info("Dropping invalid workflow", "error", err)
				continue
			}

			wflw := toWorkflow(grpcWorkflow)

			// Start a new execution context so we can cancel it as needed.
			ctx, err := idx.Insert(stream.Context(), wflw.ID)
			if err != nil {
				// Handle already excuting workflow. Perhaps this needs to be an agent concern
				// so that multiple transports benefit from the same handling. Or, given its
				// already running, perhaps we just log we were asked to run the same workflow
				// twice.
				_ = err
			}

			go func(ctx context.Context, wflw workflow.Workflow) {
				if err := handler.HandleWorkflow(ctx, wflw, g); err != nil {
					log.Info("Failed to handle workflow", "error", err)
				}

				// Stop the execution context so we're no longer tracking the workflow.
				idx.Cancel(wflw.ID)
			}(ctx, wflw)

		case *workflowproto.StreamWorkflowsResponse_StopWorkflow_:
			req := request.GetStopWorkflow()
			// TODO: Validate workflow ID
			idx.Cancel(req.WorkflowId)
		}
	}
}

func (g *GRPC) RecordEvent(ctx context.Context, e event.Event) {
	evnt, err := toGRPC(e)
	if err != nil {
		g.log.Error(err, "convert event to gRPC payload", "event", e)
		return
	}

	_, err = g.client.PublishEvent(ctx, &workflowproto.PublishEventRequest{
		Event: evnt,
	})
	if err != nil {
		g.log.Error(err, "publishing event", "event", evnt)
		return
	}
}

func validateGRPCWorkflow(wflw *workflowproto.Workflow) error {
	if wflw == nil {
		return errors.New("workflow must not be nil")
	}

	for _, action := range wflw.Actions {
		if action == nil {
			return errors.New("workflow actions must not be nil")
		}
	}

	return nil
}

func toWorkflow(wflw *workflowproto.Workflow) workflow.Workflow {
	return workflow.Workflow{
		ID:      wflw.WorkflowId,
		Actions: toActions(wflw.GetActions()),
	}
}

func toActions(a []*workflowproto.Workflow_Action) []workflow.Action {
	var actions []workflow.Action
	for _, action := range a {
		actions = append(actions, workflow.Action{
			ID:               action.GetId(),
			Name:             action.GetName(),
			Image:            action.GetImage(),
			Cmd:              action.GetCmd(),
			Args:             action.GetArgs(),
			Env:              action.GetEnv(),
			Volumes:          action.GetVolumes(),
			NetworkNamespace: action.GetNetworkNamespace(),
		})
	}
	return actions
}

func toGRPC(e event.Event) (*workflowproto.Event, error) {
	switch v := e.(type) {
	case event.ActionStarted:
		return &workflowproto.Event{
			WorkflowId: v.WorkflowID,
			Event: &workflowproto.Event_ActionStarted_{
				ActionStarted: &workflowproto.Event_ActionStarted{
					ActionId: v.ActionID,
				},
			},
		}, nil
	case event.ActionSucceeded:
		return &workflowproto.Event{
			WorkflowId: v.WorkflowID,
			Event: &workflowproto.Event_ActionSucceeded_{
				ActionSucceeded: &workflowproto.Event_ActionSucceeded{
					ActionId: v.ActionID,
				},
			},
		}, nil
	case event.ActionFailed:
		return &workflowproto.Event{
			WorkflowId: v.WorkflowID,
			Event: &workflowproto.Event_ActionFailed_{
				ActionFailed: &workflowproto.Event_ActionFailed{
					ActionId:       v.ActionID,
					FailureReason:  &v.Reason,
					FailureMessage: &v.Message,
				},
			},
		}, nil
	case event.WorkflowRejected:
		return &workflowproto.Event{
			WorkflowId: v.ID,
			Event: &workflowproto.Event_WorkflowRejected_{
				WorkflowRejected: &workflowproto.Event_WorkflowRejected{
					Message: v.Message,
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("grpc: %w", event.IncompatibleError{
		Event: e,
	})
}

type workflowIndex struct {
	cancellers map[string]context.CancelFunc
	mtx        sync.Mutex
}

func (c *workflowIndex) Insert(ctx context.Context, id string) (context.Context, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.cancellers == nil {
		c.cancellers = map[string]context.CancelFunc{}
	}

	if _, ok := c.cancellers[id]; ok {
		return nil, fmt.Errorf("workflow is already tracked (%v)", id)
	}

	// Create a new cancellation function and add it to the c
	ctx, cancel := context.WithCancel(ctx)
	c.cancellers[id] = cancel
	return ctx, nil
}

func (c *workflowIndex) Cancel(id string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.cancellers == nil {
		return
	}

	if cancel, ok := c.cancellers[id]; ok {
		cancel()
	}

	delete(c.cancellers, id)
}
