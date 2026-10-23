package planning

import (
	"github.com/jteutenberg/understate/actions"
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type PartialPlan struct {
	initialState  *state.State
	positiveGoals []*core.Predicate
	negativeGoals []*core.Predicate

	// the plan tracks steps (disjunctions of actions) and causal links between them
	steps       []*Step
	causalLinks []*CausalLink
	// open work can be:
	// any open predicate without a causal link fulfilling it
	// any causal link that is threatened by another predicate
	// any disjunctive set of actions that has at least one unshared precondition with
	//  no causal link fulfilling it

	// it also has a list of open work, but also completed work giving the decisions made
	// completed work will had an "undo" option. Depending on what was done it could be
	// 1. An ordering between two steps; the causal link that unthreatens
	// 2. A reduction of a disjuntive set; and the set it used to be
	// 3. A predicate and its step; the new causal link that fulfills it
}

// Step is an abstract action with preconditions and effects that are the
// intersection of a (disjunctive) set of actions
type Step struct {
	actions []*actions.Action
	// preconditions shared by all actions in the step
	preconditions []*core.Predicate
	// effects shared by all actions in the step
	effects []*core.Predicate
}
type CausalLink struct {
	fromStep *Step
	toStep   *Step
	// TODO: is this enough to match the predicate in fromStep to toStep?
	predicate *core.Predicate
}

func NewPartialPlan() *PartialPlan {
	return &PartialPlan{}
}
