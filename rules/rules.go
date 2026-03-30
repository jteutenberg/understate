package rules

import (
	"fmt"

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

// NewRule creates a new Rule from the RHS and LHS predicates, with their
// variable references being modified so that they are shared based on
// their labels
func NewRule(lhs *core.Predicate, rhs []*core.Predicate) *Rule {
	// share variable references by name
	sharedVars := make(map[string]*core.VariableReference)
	for _, varRef := range lhs.VarRefs {
		if existingVarRef, ok := sharedVars[varRef.Label]; ok {
			varRef.Ref = existingVarRef.Ref
		} else {
			sharedVars[varRef.Label] = varRef
		}
	}
	for _, rhsPred := range rhs {
		for _, varRef := range rhsPred.VarRefs {
			if existingVarRef, ok := sharedVars[varRef.Label]; ok {
				varRef.Ref = existingVarRef.Ref
			} else {
				sharedVars[varRef.Label] = varRef
			}
		}
	}
	return &Rule{
		lhs: lhs,
		rhs: rhs,
	}
}

func (rm *RuleMachine) AddRule(rule *Rule) {
	rm.rules = append(rm.rules, rule)
}

func (rm *RuleMachine) Answer(p *core.Predicate) <-chan []*core.Atomic {
	answer := make(chan []*core.Atomic)
	go func() {
		for _, rule := range rm.rules {
			if rule.lhs.CanUnify(p) {
				unified := rule.lhs.Clone()
				err := unified.Unify(p)
				if err != nil {
					fmt.Printf("error unifying rule %v with %v: %v", rule.lhs, p, err)
					continue
				}
				// The RHS predicates should be sharing variable references.
				// In which case those are unified with p already

				// get an answer for each RHS predicate and
				// begin DFS
			}
		}
		close(answer)
	}()
	return answer
}
