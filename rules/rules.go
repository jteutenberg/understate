package rules

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type RuleMachine struct {
	core.Answerer
	rules       []*Rule
	subAnswerer core.Answerer
	state       *state.State // for caching intermediate results
}

type Rule struct {
	lhs *core.Predicate
	rhs []*core.Predicate
}

var Cut = &core.Predicate{
	Definition: &core.PredicateDefinition{
		Functor: "cut",
	},
}

func NewRuleMachine(subAnswerer core.Answerer, state *state.State) *RuleMachine {
	return &RuleMachine{
		subAnswerer: subAnswerer,
		state:       state,
		rules:       make([]*Rule, 0, 10),
	}
}

func (rm *RuleMachine) Answer(p *core.Predicate) <-chan []*core.Atomic {
	answer := make(chan []*core.Atomic)
	go func() {
		for _, rule := range rm.rules {
			if rule.lhs.CanUnify(p) {
				// need to globally unify with each RHS predicate
				// get an answer for each RHS predicate and
				// begin DFS
			}
		}
		close(answer)
	}()
	return answer
}
