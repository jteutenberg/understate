package state

import "github.com/jteutenberg/understate/core"

// Stacked states, where the top state accepts fact updates, but all
// will provide answers (if prior states return unknown)
type StackedState struct {
	states []State
}

// Creates a StackedState with the baseState on the bottom and an empty state on top
func NewStackedState(baseState *simpleState) *StackedState {
	states := make([]State, 0)
	// create a top state that shares all atomics and types, but has no facts
	topState := &simpleState{
		AllAtomics:     baseState.AllAtomics,
		Types:          baseState.Types,
		atomicsByName:  baseState.atomicsByName,
		atomicsByIndex: baseState.atomicsByIndex,
		numericAtomics: baseState.numericAtomics,
	}
	states = append(states, topState)
	// add the base state to the bottom of the stack
	states = append(states, baseState)
	return &StackedState{
		states: states,
	}
}

func (s *StackedState) Answer(p *core.Predicate, frame *core.Frame, ctx core.QueryContext, history *core.SearchHistory) <-chan *core.Predicate {
	answers := make(chan *core.Predicate)
	go func() {
		defer close(answers)
		for _, state := range s.states {
			subanswers := state.Answer(p, frame, ctx, history)
			// if there is an answer, forward it on and return
			for answer := range subanswers {
				answers <- answer
				if answer != nil {
					return
				}
			}
		}
	}()
	return answers
}

// Other possible states (future work)
// - States that split queries and updates by predicate definition
