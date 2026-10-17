package actions

import (
	"github.com/jteutenberg/understate/core"
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

// ApplicableActions finds all valid unifications with each action's signature.
func (as *ActionSet) ApplicableActions(answerer core.Answerer) <-chan *Action {
	output := make(chan *Action)
	go func() {
		defer close(output)
		for _, action := range as.actions {
			groundActions := action.GetApplicableActions(answerer)
			for groundAction := range groundActions {
				output <- groundAction
			}
		}
	}()
	return output
}
