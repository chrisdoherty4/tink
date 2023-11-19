package queue

import (
	"container/heap"

	"github.com/tinkerbell/tink/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkflowQueue struct {
	heap []*v1alpha2.Workflow
}

func (wq WorkflowQueue) Pop() *v1alpha2.Workflow {

}

func (wq WorkflowQueue) Push(o client.Object) {

}

type workflowHeap []*v1alpha2.Workflow

func (pq workflowHeap) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq workflowHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *workflowHeap) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *workflowHeap) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *workflowHeap) update(item *Item, value string, priority int) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}
