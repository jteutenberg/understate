package rules

import (
	"fmt"
	"strconv"

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
	lhs        *core.Predicate
	rhs        []*core.Predicate
	sharedVars map[string]*core.VariableReference
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
	for i, varRef := range lhs.VarRefs {
		varLabel := varRef.Label
		if existingVarRef, ok := sharedVars[varLabel]; ok {
			// same label, use the same variable reference
			lhs.VarRefs[i] = existingVarRef
		} else {
			sharedVars[varLabel] = varRef
		}
	}
	for _, rhsPred := range rhs {
		for i, varRef := range rhsPred.VarRefs {
			if existingVarRef, ok := sharedVars[varRef.Label]; ok {
				rhsPred.VarRefs[i] = existingVarRef
			} else {
				sharedVars[varRef.Label] = varRef
			}
		}
	}
	return &Rule{
		lhs:        lhs,
		rhs:        rhs,
		sharedVars: sharedVars,
	}
}

func cloneSharedVars(sharedVars map[string]*core.VariableReference) map[string]*core.VariableReference {
	newSharedVars := make(map[string]*core.VariableReference)
	for label, varRef := range sharedVars {
		newSharedVars[label] = varRef.Clone().(*core.VariableReference)
	}
	return newSharedVars
}

func (r *Rule) Clone() *Rule {
	// new share variables map
	sharedVars := cloneSharedVars(r.sharedVars)
	newRHS := make([]*core.Predicate, len(r.rhs))
	for i, rhsPred := range r.rhs {
		newRHS[i] = rhsPred.CloneWithVars(sharedVars)
	}
	rule := &Rule{
		lhs:        r.lhs.CloneWithVars(sharedVars),
		rhs:        newRHS,
		sharedVars: sharedVars,
	}

	return rule
}

func (r *Rule) avoidDuplicates(p *core.Predicate) {
	// check each variable name in the predicate
	for _, varRef := range p.VarRefs {
		if varRef.Dereference().Ref == nil {
			// a variable
			if r.sharedVars[varRef.Label] != nil {
				// rename to avoid duplicates after unification
				vr := r.sharedVars[varRef.Label]
				vr.Label = vr.Label + "_" + strconv.Itoa(len(r.sharedVars))
				r.sharedVars[vr.Label] = vr
				delete(r.sharedVars, varRef.Label)
			}
		}
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
				unified := rule.Clone()
				unified.avoidDuplicates(p)
				err := unified.lhs.Unify(p)
				if err != nil {
					fmt.Printf("error unifying rule %v with %v: %v", rule.lhs, p, err)
					continue
				}
				rm.checkAnswers(unified.rhs, unified.sharedVars)
			}
		}
		close(answer)
	}()
	return answer
}

func (rm *RuleMachine) checkAnswers(rhs []*core.Predicate, sharedVars map[string]*core.VariableReference) {
	stack := make([]<-chan []*core.Atomic, 0, len(rhs))
	stack = append(stack, rm.subAnswerer.Answer(rhs[0]))
	varStack := make([]map[string]*core.VariableReference, 0, len(rhs))
	varStack = append(varStack, sharedVars)
	rhsStack := make([][]*core.Predicate, 0, len(rhs))
	rhsStack = append(rhsStack, rhs)
	for {
		ans := <-stack[len(stack)-1]
		//fmt.Println("Answer at depth", len(stack), "is", ans)
		if ans == nil {
			// no more answers
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				// we're done
				return
			}
			rhsStack = rhsStack[:len(rhsStack)-1]
			// backtrack to the previous RHS version
			//fmt.Println("Backtrack to previous rule", len(stack))
			continue
		}
		// so now we need to unify this answer with the current rule, and continue

		// NOTE: if the next rule is a fact already, we can skip over the cloning bit
		// as there is no need to unify the arguments

		// clone the current state, ready to update
		nextSharedVars := cloneSharedVars(varStack[len(varStack)-1])
		nextRHS := make([]*core.Predicate, len(rhsStack[len(rhsStack)-1]))
		for i, nextPred := range rhsStack[len(rhsStack)-1] {
			nextRHS[i] = nextPred.CloneWithVars(nextSharedVars)
		}
		// unify returned atomics
		//fmt.Println("Working on answers for", nextRHS[len(stack)-1])
		unifies := true
		for i, varRef := range nextRHS[len(stack)-1].VarRefs {
			if varRef.CanUnify(ans[i]) {
				varRef.Unify(ans[i])
			} else {
				// try the next answer instead
				unifies = false
				//fmt.Println("Cannot unify", varRef, "with", ans[i])
				break
			}
		}
		if !unifies {
			continue
		}
		//fmt.Println("After unifying with last rule answer:", nextRHS[len(stack)-1])
		if len(stack) == len(rhs) {
			// TODO: yield the answer
			fmt.Println("Got an answer!", nextSharedVars)
		} else {
			// recurse
			//fmt.Println("Move down to next rule", len(stack))
			stack = append(stack, rm.subAnswerer.Answer(nextRHS[len(stack)]))
			varStack = append(varStack, nextSharedVars)
			rhsStack = append(rhsStack, nextRHS)
		}
	}
}
