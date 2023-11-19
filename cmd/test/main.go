package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/tinkerbell/tink/api/v1alpha2"
	"github.com/tinkerbell/tink/internal/server/index"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer cancel()

	ccfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: "kubeconfig"},
		nil)

	cfg, err := ccfg.ClientConfig()
	if err != nil {
		panic(err)
	}

	clstr, err := cluster.New(cfg)
	if err != nil {
		panic(err)
	}

	if err := clstr.GetFieldIndexer().IndexField(ctx, &v1alpha2.Workflow{}, index.WorkflowAgentID, index.WorkflowAgentIDFn); err != nil {
		panic(err)
	}

	clnt := clstr.GetClient()

	var wl v1alpha2.WorkflowList
	watch, err := clnt.Watch(ctx, &wl, client.MatchingFields{
		index.WorkflowAgentID: "",
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
}
