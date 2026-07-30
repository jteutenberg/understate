package actions

import (
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type Action struct {
	Signature             *core.Predicate
	allPreconditions      []*core.Predicate
	PositivePreconditions []*core.Predicate
	NegativePreconditions []*core.Predicate
	AddEffects            []*core.Predicate
	DeleteEffects         []*core.Predicate
	frame                 *core.Frame
}

func NewAction(signature *core.Predicate, preconditions []*core.Predicate, effects []*core.Predicate, frame *core.Frame) *Action {

	// any not() preconditions can be split off into a negative precondition set
	positivePreconditions := make([]*core.Predicate, 0, len(preconditions))
	negativePreconditions := make([]*core.Predicate, 0, len(preconditions))
	for _, precond := range preconditions {
		if precond.Definition.Functor == "not" {
			negativePreconditions = append(negativePreconditions, precond.VarRefs[0].Ref.(*core.Predicate))
		} else {
			positivePreconditions = append(positivePreconditions, precond)
		}
	}
	// and effects
	addEffects := make([]*core.Predicate, 0, len(effects))
	deleteEffects := make([]*core.Predicate, 0, len(effects))
	for _, effect := range effects {
		if effect.Definition.Functor == "not" {
			deleteEffects = append(deleteEffects, effect.VarRefs[0].Ref.(*core.Predicate))
		} else {
			addEffects = append(addEffects, effect)
		}
	}
	return &Action{
		Signature:             signature,
		allPreconditions:      preconditions,
		PositivePreconditions: positivePreconditions,
		NegativePreconditions: negativePreconditions,
		AddEffects:            addEffects,
		DeleteEffects:         deleteEffects,
		frame:                 frame,
	}
}

// Applicable when all preconditions are met according to the given answerer
func (a *Action) IsApplicable(ans core.Answerer) bool {
	// test the conjunction of all preconditions
	if len(a.allPreconditions) > 0 {
		answers := core.AnswerConjunction(ans, a.allPreconditions, a.frame, core.NewQueryContext())
		answer := <-answers
		if answer == nil {
			return false
		}
	}
	return true
}

// Return ground versions of this action that are applicable to the given answerer
func (a *Action) GetApplicableActions(ans core.Answerer) []*Action {
	return nil
}

// ApplyTo updates any ground effects of this action to the given state
func (a *Action) ApplyTo(s *state.State) {
	for _, addEffect := range a.AddEffects {
		if addEffect.IsFact() {
			s.SetTrue(addEffect)
		}
	}
	for _, deleteEffect := range a.DeleteEffects {
		if deleteEffect.IsFact() {
			s.SetFalse(deleteEffect)
		}
	}
}

func (a *Action) Clone() *Action {
	frame := a.frame.Clone()
	signature := a.Signature.CloneInFrame(frame)
	act := Action{
		Signature:             signature,
		allPreconditions:      make([]*core.Predicate, len(a.allPreconditions)),
		PositivePreconditions: make([]*core.Predicate, len(a.PositivePreconditions)),
		NegativePreconditions: make([]*core.Predicate, len(a.NegativePreconditions)),
		AddEffects:            make([]*core.Predicate, len(a.AddEffects)),
		DeleteEffects:         make([]*core.Predicate, len(a.DeleteEffects)),
		frame:                 frame,
	}
	for i, precond := range a.PositivePreconditions {
		act.PositivePreconditions[i] = precond.CloneInFrame(frame)
		act.allPreconditions[i] = act.PositivePreconditions[i]
	}
	for i, precond := range a.NegativePreconditions {
		act.NegativePreconditions[i] = precond.CloneInFrame(frame)
		allIndex := i + len(a.PositivePreconditions)
		act.allPreconditions[allIndex] = a.allPreconditions[allIndex].CloneInFrame(frame)
	}
	for i, effect := range a.AddEffects {
		act.AddEffects[i] = effect.CloneInFrame(frame)
	}
	for i, effect := range a.DeleteEffects {
		act.DeleteEffects[i] = effect.CloneInFrame(frame)
	}
	return &act
}
