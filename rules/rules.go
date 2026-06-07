package rules

import (
	"context"
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
	lhs   *core.Predicate
	rhs   []*core.Predicate
	frame *core.Frame
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
// the frame passed in should have been used to create all LHS and RHS predicates
func NewRule(lhs *core.Predicate, rhs []*core.Predicate, frame *core.Frame) *Rule {
	return &Rule{
		lhs:   lhs.CloneInFrame(frame),
		rhs:   rhs,
		frame: frame,
	}
}

func (r *Rule) Clone() *Rule {
	// new share variables map
	frame := r.frame.Clone()
	newRHS := make([]*core.Predicate, len(r.rhs))
	for i, rhsPred := range r.rhs {
		newRHS[i] = rhsPred.CloneInFrame(frame)
	}
	rule := &Rule{
		lhs:   r.lhs.CloneInFrame(frame),
		rhs:   newRHS,
		frame: frame,
	}

	return rule
}

func (r *Rule) avoidDuplicates(p *core.Predicate) {
	// check each variable name in the predicate
	for _, varRef := range p.VarRefs {
		if varRef.Dereference().Ref == nil {
			// a variable
			if r.frame.Vars[varRef.Label] != nil {
				// rename to avoid duplicates after unification
				oldLabel := varRef.Label
				vr := r.frame.Vars[oldLabel]
				vr.Label = oldLabel + "_" + strconv.Itoa(len(r.frame.Vars))
				r.frame.Vars[vr.Label] = vr
				delete(r.frame.Vars, oldLabel)
			}
		}
	}
}

func (rm *RuleMachine) AddRule(rule *Rule) {
	rm.rules = append(rm.rules, rule)
}

func (rm *RuleMachine) Answer(p *core.Predicate, frame *core.Frame, ctx context.Context) <-chan *core.Predicate {
	answers := make(chan *core.Predicate)
	go func() {
	loopRules:
		for _, rule := range rm.rules {
			// Note: conflicts between frame (from predicate) with rules frame
			// are sorted below with unified.avoidDuplicates()
			if rule.lhs.CanUnify(p) {
				unified := rule.Clone()
				unified.avoidDuplicates(p)
				err := unified.lhs.Unify(p)
				if err != nil {
					fmt.Printf("error unifying rule %v with %v: %v", rule.lhs, p, err)
					continue
				}
				rm.checkAnswers(unified, answers, ctx)
				select {
				case <-ctx.Done():
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

func (rm *RuleMachine) checkAnswers(rule *Rule, answers chan<- *core.Predicate, ctx context.Context) {
	stack := make([]<-chan *core.Predicate, 0, len(rule.rhs))
	// because we are passing the frame, do we need to clone the rule?
	// No. It was cloned just before calling
	stack = append(stack, rm.subAnswerer.Answer(rule.rhs[0], rule.frame, ctx))
	ruleStack := make([]*Rule, 0, len(rule.rhs))
	ruleStack = append(ruleStack, rule)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// continue
		}
		ans := <-stack[len(stack)-1]
		if ans == core.Terminate || ans == nil {
			// no more answers
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				// we're done
				return
			}
			ruleStack = ruleStack[:len(ruleStack)-1]
			// backtrack to the previous RHS version
			continue
		}
		// so now we need to unify this answer with the current rule, and continue
		// NOTE: if the next rule is a fact already, we can skip over the cloning bit
		// as there is no need to unify the arguments
		if !ruleStack[len(stack)-1].rhs[len(stack)-1].CanUnify(ans) {
			continue
		}
		nextRule := ruleStack[len(stack)-1].Clone()
		nextRule.rhs[len(stack)-1].Unify(ans)

		if len(stack) == len(rule.rhs) {
			answers <- nextRule.lhs
		} else {
			// recurse
			stack = append(stack, rm.subAnswerer.Answer(nextRule.rhs[len(stack)], nextRule.frame, ctx))
			ruleStack = append(ruleStack, nextRule)
		}
	}
}
