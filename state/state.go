package state

import (
	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/core"
)

type State struct {
	core.Answerer
	// facts by Functor name
	TrueFacts  map[string][]*core.Predicate
	FalseFacts map[string][]*core.Predicate
}

func (s *State) Answer(p *core.Predicate) (<-chan []*core.Atomic, []*bitset.IntSet) {
	trueFacts := s.TrueFacts[p.Definition.Functor]
	falseFacts := s.FalseFacts[p.Definition.Functor]
	answer := make(chan []*core.Atomic)
	go func() {
		// if this is a ground predicate and is in falseFacts, then
		// send a Terminate token and close the channel
		isFact := p.IsFact()
		if isFact {
			for _, fact := range falseFacts {
				if fact.CanUnify(p) {
					answer <- []*core.Atomic{&core.Terminate}
					close(answer)
					return
				}
			}
		}
		for _, fact := range trueFacts {
			if p.CanUnify(fact) {
				// then return this fact's values
				values := make([]*core.Atomic, len(p.VarRefs))
				for i, varRef := range fact.VarRefs {
					values[i] = varRef.Dereference().Ref.(*core.Atomic)
				}
				answer <- values
				if isFact {
					break
				}
			}
		}
		close(answer)
	}()
	maxSets := make([]*bitset.IntSet, len(p.VarRefs))
	for i, varDef := range p.Definition.ArgDefinitions {
		maxSets[i] = varDef.Type.Atomics
	}
	return answer, maxSets
}
