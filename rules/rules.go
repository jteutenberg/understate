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

func (rm *RuleMachine) Answer(p *core.Predicate, halt <-chan bool) <-chan *core.Predicate {
	answers := make(chan *core.Predicate)
	go func() {
	loopRules:
		for _, rule := range rm.rules {
			if rule.lhs.CanUnify(p) {
				unified := rule.Clone()
				unified.avoidDuplicates(p)
				err := unified.lhs.Unify(p)
				if err != nil {
					fmt.Printf("error unifying rule %v with %v: %v", rule.lhs, p, err)
					continue
				}
				rm.checkAnswers(unified, answers, halt)
				select {
				case <-halt:
					break loopRules
				default:
					// continue
				}
			}
		}
		close(answers)
	}()
	return answers
}

func (rm *RuleMachine) String() string {
	return "A rule machine"
}

func (rm *RuleMachine) checkAnswers(rule *Rule, answers chan<- *core.Predicate, halt <-chan bool) {
	haltStack := make([]chan bool, 0, len(rule.rhs))
	haltStack = append(haltStack, make(chan bool))
	stack := make([]<-chan *core.Predicate, 0, len(rule.rhs))
	stack = append(stack, rm.subAnswerer.Answer(rule.rhs[0], haltStack[0]))
	ruleStack := make([]*Rule, 0, len(rule.rhs))
	ruleStack = append(ruleStack, rule)
	for {
		select {
		case <-halt:
			// terminate all sub-answerers
			for _, halt := range haltStack {
				close(halt)
			}
			return
		default:
			// continue
		}
		ans := <-stack[len(stack)-1]
		//fmt.Println("Answer at depth", len(stack), "is", ans)
		if ans == core.Terminate || ans == nil {
			// no more answers
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				// we're done
				return
			}
			ruleStack = ruleStack[:len(ruleStack)-1]
			haltStack = haltStack[:len(haltStack)-1]
			// backtrack to the previous RHS version
			//fmt.Println("Backtrack to previous rule", len(stack))
			continue
		}
		// so now we need to unify this answer with the current rule, and continue

		// NOTE: if the next rule is a fact already, we can skip over the cloning bit
		// as there is no need to unify the arguments

		// unify returned atomics
		//fmt.Println("Working on answers for", nextRHS[len(stack)-1])
		if !ruleStack[len(stack)-1].rhs[len(stack)-1].CanUnify(ans) {
			continue
		}
		nextRule := ruleStack[len(stack)-1].Clone()
		nextRule.rhs[len(stack)-1].Unify(ans)

		if len(stack) == len(rule.rhs) {
			answers <- nextRule.lhs
		} else {
			// recurse
			//fmt.Println("Move down to next rule", len(stack))
			haltStack = append(haltStack, make(chan bool))
			stack = append(stack, rm.subAnswerer.Answer(nextRule.rhs[len(stack)], haltStack[len(haltStack)-1]))
			ruleStack = append(ruleStack, nextRule)
		}
	}
}
