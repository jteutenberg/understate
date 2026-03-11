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

	AllAtomics    *bitset.IntSet
	atomicsByName map[string]*core.Atomic
	Types         map[string]*core.Type
}

func NewState() *State {
	return &State{
		TrueFacts:     make(map[string][]*core.Predicate),
		FalseFacts:    make(map[string][]*core.Predicate),
		AllAtomics:    bitset.NewIntSet(),
		atomicsByName: make(map[string]*core.Atomic),
		Types:         make(map[string]*core.Type),
	}
}

func (s *State) Answer(p *core.Predicate) <-chan []*core.Atomic {
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
	return answer
}

func (s *State) GetAtomic(name string, t *core.Type) *core.Atomic {
	if a := s.atomicsByName[name]; a != nil {
		return a
	}
	atomicIndex := uint(0)
	if t != nil {
		var ok bool
		ok, atomicIndex = t.Atomics.GetLastValue()
		atomicIndex++
		if !ok || s.AllAtomics.Contains(atomicIndex) {
			_, atomicIndex = s.AllAtomics.GetLastValue()
			atomicIndex += 5
		}
	} else {
		_, atomicIndex = s.AllAtomics.GetLastValue()
		atomicIndex += 1
	}
	atomic := &core.Atomic{
		Index: int(atomicIndex),
		Value: name,
		Type:  t,
	}
	s.AllAtomics.Add(atomicIndex)
	s.atomicsByName[name] = atomic
	if t != nil {
		t.Atomics.Add(atomicIndex)
	}
	return atomic
}

func (s *State) SetTrue(p *core.Predicate) {
	// TODO: assert that the predicate is a fact
	if s.TrueFacts[p.Definition.Functor] == nil {
		s.TrueFacts[p.Definition.Functor] = make([]*core.Predicate, 0)
	}
	//TODO: just return if the predicate is already in the list
	s.TrueFacts[p.Definition.Functor] = append(s.TrueFacts[p.Definition.Functor], p)
}

func (s *State) SetFalse(p *core.Predicate) {
	// TODO: assert that the predicate is a fact
	if s.FalseFacts[p.Definition.Functor] == nil {
		s.FalseFacts[p.Definition.Functor] = make([]*core.Predicate, 0)
	}
	s.FalseFacts[p.Definition.Functor] = append(s.FalseFacts[p.Definition.Functor], p)
	//TODO: remove from true facts, if it exists
}
