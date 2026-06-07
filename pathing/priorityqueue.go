package pathing

import "container/heap"

type priorityQueue []*searchNode

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	// min heap: lowest cost first
	return pq[i].cost < pq[j].cost
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	node := x.(*searchNode)
	node.index = len(*pq)
	*pq = append(*pq, node)
}

func (pq *priorityQueue) Pop() interface{} {
	n := len(*pq)
	item := (*pq)[n-1]
	(*pq)[n-1] = nil // remove reference from the array
	*pq = (*pq)[:n-1]
	return item
}

func (pq *priorityQueue) reducedCost(node *searchNode) {
	heap.Fix(pq, node.index)
}
