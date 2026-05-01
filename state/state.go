package state

import (
	"strconv"

	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/core"
)

// positive integers
var Numeric = &core.Type{
	Name:    "Numeric",
	Atomics: bitset.NewIntSet(),
}

type State struct {
	core.Answerer
	// facts by Functor name
	TrueFacts  map[string][]*core.Predicate
	FalseFacts map[string][]*core.Predicate

	AllAtomics    *bitset.IntSet
	atomicsByName map[string]*core.Atomic
	Types         map[string]*core.Type

	numericAtomics []*core.Atomic
}

func NewState() *State {
	return &State{
		TrueFacts:      make(map[string][]*core.Predicate),
		FalseFacts:     make(map[string][]*core.Predicate),
		AllAtomics:     bitset.NewIntSet(),
		atomicsByName:  make(map[string]*core.Atomic),
		Types:          make(map[string]*core.Type),
		numericAtomics: make([]*core.Atomic, 1000000),
	}
}

func (s *State) Answer(p *core.Predicate, halt <-chan bool) <-chan *core.Predicate {
	trueFacts := s.TrueFacts[p.Definition.Functor]
	falseFacts := s.FalseFacts[p.Definition.Functor]
	answers := make(chan *core.Predicate)
	go func() {
		// if this is a ground predicate and is in falseFacts, then
		// send a Terminate token and close the channel
		isFact := p.IsFact()
		if isFact {
			for _, fact := range falseFacts {
				if fact.CanUnify(p) {
					answers <- core.Terminate
					close(answers)
					goto done
				}
			}
		}
		for _, fact := range trueFacts {
			if p.CanUnify(fact) {
				answers <- fact
				if isFact {
					break
				}
				//if halt has been closed, end now
				select {
				case <-halt:
					goto done
				default:
					// continue
				}
			}
		}
	done:
		close(answers)
	}()
	return answers
}

func (s *State) String() string {
	return "A state"
}

func (s *State) GetNumericAtomic(index int) *core.Atomic {
	if index < 0 {
		return nil
	}

	if s.numericAtomics[index] == nil || index >= len(s.numericAtomics) {
		a := &core.Atomic{
			Index: index,
			Value: strconv.Itoa(index),
			Type:  Numeric,
		}
		if index < len(s.numericAtomics) {
			s.numericAtomics[index] = a
		} else {
			s.numericAtomics = append(s.numericAtomics, a)
		}
		return a
	}
	return s.numericAtomics[index]
}

func (s *State) GetAtomic(name string, t *core.Type) *core.Atomic {
	if a := s.atomicsByName[name]; a != nil {
		return a
	}
	if t == Numeric {
		// value from name
		atomicIndex, err := strconv.Atoi(name)
		if err != nil || atomicIndex < 0 {
			return nil
		}
		return s.GetNumericAtomic(atomicIndex)
	}
	//handle non-numeric atomics
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
