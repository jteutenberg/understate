package actions

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type Action struct {
	Name                  string
	PositivePreconditions []*core.Predicate
	NegativePreconditions []*core.Predicate
	AddEffects            []*core.Predicate
	DeleteEffects         []*core.Predicate
	frame                 *core.Frame
}

func (a *Action) IsApplicable(ans *core.Answerer) bool {
	return false
}

// Return ground versions of this action that are applicable to the given answerer
func (a *Action) GetApplicableActions(ans *core.Answerer) []*Action {
	return nil
}

// ApplyTo updates any ground effects of this action to the given state
func (a *Action) ApplyTo(s *state.State) {
}

func (a *Action) Clone() *Action {
	act := Action{
		Name:                  a.Name,
		PositivePreconditions: make([]*core.Predicate, len(a.PositivePreconditions)),
		NegativePreconditions: make([]*core.Predicate, len(a.NegativePreconditions)),
		AddEffects:            make([]*core.Predicate, len(a.AddEffects)),
		DeleteEffects:         make([]*core.Predicate, len(a.DeleteEffects)),
	}
	frame := a.frame.Clone()
	for i, precond := range a.PositivePreconditions {
		act.PositivePreconditions[i] = precond.CloneInFrame(frame)
	}
	for i, precond := range a.NegativePreconditions {
		act.NegativePreconditions[i] = precond.CloneInFrame(frame)
	}
	for i, effect := range a.AddEffects {
		act.AddEffects[i] = effect.CloneInFrame(frame)
	}
	for i, effect := range a.DeleteEffects {
		act.DeleteEffects[i] = effect.CloneInFrame(frame)
	}
	return &act
}
