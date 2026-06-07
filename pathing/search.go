package pathing

import (
	"container/heap"
	"context"

	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/core"
)

type Heuristic interface {
	AtomicType() *core.Type
	Estimate(from *core.Atomic, to *core.Atomic) uint
}
type Search struct {
	heuristics []Heuristic
	answerer   core.Answerer
}
type searchNode struct {
	atomic *core.Atomic
	cost   uint
	prev   *searchNode
	index  int
}

func (node *searchNode) traceBack() []*core.Atomic {
	path := make([]*core.Atomic, 0, 100)
	for node != nil {
		path = append(path, node.atomic)
		node = node.prev
	}
	return path
}

func (search *Search) ShortestPath(start *core.Atomic, end *core.Atomic, connections *core.PredicateDefinition) []*core.Atomic {
	var h Heuristic
	for _, heuristic := range search.heuristics {
		if heuristic.AtomicType() == start.Type {
			h = heuristic
			break
		}
	}

	var connectTemplate *core.Predicate

	if len(connections.ArgDefinitions) == 3 {
		connectTemplate = &core.Predicate{
			Definition: connections,
			VarRefs: []*core.VariableReference{
				{Label: "From", Ref: nil},
				{Label: "To", Ref: nil},
				{Label: "Cost", Ref: nil},
			},
		}
	} else if len(connections.ArgDefinitions) == 2 {
		connectTemplate = &core.Predicate{
			Definition: connections,
			VarRefs: []*core.VariableReference{
				{Label: "From", Ref: nil},
				{Label: "To", Ref: nil},
			},
		}
	} else {
		return nil
	}

	ctx := context.Background()

	open := priorityQueue(make([]*searchNode, 0, 100))

	heap.Push(&open, &searchNode{
		atomic: start,
		cost:   0,
		prev:   nil,
	})

	visited := bitset.NewIntSet()
	for len(open) > 0 {
		current := heap.Pop(&open).(*searchNode)
		if current.atomic.Index == end.Index {
			return current.traceBack()
		}
		visited.Add(current.atomic.Index)
		query := connectTemplate.Clone().(*core.Predicate)
		query.VarRefs[0].Ref = current.atomic
		adjacent := search.answerer.Answer(query, nil, ctx)
		for adjacent := range adjacent {
			adjAtomic := adjacent.VarRefs[1].Ref.(*core.Atomic)
			if adjacent == core.Terminate || visited.Contains(adjAtomic.Index) {
				continue
			}
			var cost uint = 0
			if len(adjacent.VarRefs) == 3 {
				cost += adjacent.VarRefs[2].Ref.(*core.Atomic).Index
			} else {
				cost += 1
			}
			if h != nil {
				cost += h.Estimate(adjAtomic, end)
			}
			heap.Push(&open, &searchNode{
				atomic: adjAtomic,
				cost:   current.cost + cost,
				prev:   current,
			})
		}
	}
	return nil

}
