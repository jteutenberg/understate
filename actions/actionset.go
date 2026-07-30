package actions

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type ActionSet struct {
	actions []*Action
}

func NewActionSet() *ActionSet {
	return &ActionSet{
		actions: make([]*Action, 0),
	}
}

func (as *ActionSet) AddAction(action *Action) {
	as.actions = append(as.actions, action)
}

func (as *ActionSet) GetAction(signature *core.Predicate) *Action {
	for _, action := range as.actions {
		if action.Signature.CanUnify(signature) {
			act := action.Clone()
			act.Signature.Unify(signature)
			return act
		}
	}
	return nil
}

// ApplicableActions uses a state to generate possible atomics for negative preconditions, and an
// answerer to find all valid unifications with each action's signature.
func (as *ActionSet) ApplicableActions(state *state.State, answerer core.Answerer) <-chan *Action {
	return nil
}
