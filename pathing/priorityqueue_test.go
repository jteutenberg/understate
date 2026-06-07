package pathing

import (
	"container/heap"
	"testing"

	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

func TestPriorityQueue(t *testing.T) {
	pq := priorityQueue(make([]*searchNode, 0, 100))
	heap.Push(&pq, &searchNode{atomic: &core.Atomic{Index: 1, Type: state.Numeric}, cost: 2, prev: nil})
	heap.Push(&pq, &searchNode{atomic: &core.Atomic{Index: 2, Type: state.Numeric}, cost: 1, prev: nil})
	heap.Push(&pq, &searchNode{atomic: &core.Atomic{Index: 3, Type: state.Numeric}, cost: 3, prev: nil})
	if pq.Len() != 3 {
		t.Errorf("priority queue length should be 3, got %d", pq.Len())
	}
	node := heap.Pop(&pq).(*searchNode)
	if node.atomic.Index != 2 {
		t.Errorf("priority queue should pop 2, got %d", node.atomic.Index)
	}
	node = heap.Pop(&pq).(*searchNode)
	if node.atomic.Index != 1 {
		t.Errorf("priority queue should pop 1, got %d", node.atomic.Index)
	}
	node = heap.Pop(&pq).(*searchNode)
	if node.atomic.Index != 3 {
		t.Errorf("priority queue should pop 3, got %d", node.atomic.Index)
	}
	if pq.Len() != 0 {
		t.Errorf("priority queue length should be 0, got %d", pq.Len())
	}
}
