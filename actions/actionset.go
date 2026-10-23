package actions

import (
	"github.com/jteutenberg/understate/core"
)

type ActionSet struct {
	actions                 []*Action
	actionsByPositiveEffect map[string][]*Action
	actionsByNegativeEffect map[string][]*Action
}

func NewActionSet() *ActionSet {
	return &ActionSet{
		actions:                 make([]*Action, 0),
		actionsByPositiveEffect: make(map[string][]*Action),
		actionsByNegativeEffect: make(map[string][]*Action),
	}
}

func (as *ActionSet) AddAction(action *Action) {
	as.actions = append(as.actions, action)
	for _, effect := range action.AddEffects {
		label := effect.Definition.Functor
		if as.actionsByPositiveEffect[label] == nil {
			as.actionsByPositiveEffect[label] = make([]*Action, 0)
		}
		as.actionsByPositiveEffect[label] = append(as.actionsByPositiveEffect[label], action)
	}
	for _, effect := range action.DeleteEffects {
		label := effect.Definition.Functor
		if as.actionsByNegativeEffect[label] == nil {
			as.actionsByNegativeEffect[label] = make([]*Action, 0)
		}
		as.actionsByNegativeEffect[label] = append(as.actionsByNegativeEffect[label], action)
	}
}

// Gets a partially or fully ground action after unifying
// the action signature with the given Predicate
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

// Gets gound actions after unifying one matching effect with the given Predicate
func (as *ActionSet) GetActionsByPositiveEffect(effect *core.Predicate) []*Action {
	label := effect.Definition.Functor
	results := make([]*Action, 0)
	for _, action := range as.actionsByPositiveEffect[label] {
		for i, posEffect := range action.AddEffects {
			if posEffect.CanUnify(effect) {
				act := action.Clone()
				act.AddEffects[i].Unify(effect)
				results = append(results, act)
			}
		}
	}
	return results
}

// Gets gound actions after unifying one matching effect with the given Predicate
func (as *ActionSet) GetActionsByNegativeEffect(effect *core.Predicate) []*Action {
	label := effect.Definition.Functor
	results := make([]*Action, 0)
	for _, action := range as.actionsByNegativeEffect[label] {
		for i, negEffect := range action.DeleteEffects {
			if negEffect.CanUnify(effect) {
				act := action.Clone()
				act.DeleteEffects[i].Unify(effect)
				results = append(results, act)
			}
		}
	}
	return results
}
