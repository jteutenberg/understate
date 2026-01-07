package state

import (
	"github.com/jteutenberg/understate/core"
)

type State struct {
	Answerer
	// facts by Functor name
	TrueFacts  map[string][]*Predicate
	FalseFacts map[string][]*Predicate
}

func (s *State) Answer(p *Predicate) (<-chan []*Atomic, []*AtomicSet) {
	trueFacts := s.TrueFacts[p.Definition.Functor]
	falseFacts := s.FalseFacts[p.Definition.Functor]
	answer := make(chan []*Atomic)
	go func() {
		// if this is a ground predicate and is in falseFacts, then
		// send a Terminate token and close the channel
		if p.IsFact() {
			for _, fact := range falseFacts {
				if fact.CanUnify(p) {
					answer <- []*Atomic{&Terminate}
					close(answer)
					return
				}
			}
		}
		for _, fact := range trueFacts {
			if p.CanUnify(fact) {
				// then return this fact's values
				values := make([]*Atomic, len(p.VarRefs))
				for i, varRef := range fact.VarRefs {
					values[i] = varRef.Dereference().Ref.(*Atomic)
				}
				answer <- values
			}
		}
		close(answer)
	}()
	maxSets := make([]*AtomicSet, len(p.VarRefs))
	for i, varDef := range p.Definition.ArgDefinitions {
		maxSets[i] = &varDef.Type.Atomics
	}
	return answer, maxSets
}
