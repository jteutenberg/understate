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

	AllAtomics     *bitset.IntSet
	atomicsByName  map[string]*core.Atomic
	atomicsByIndex map[uint]*core.Atomic
	Types          map[string]*core.Type

	numericAtomics []*core.Atomic
}

func NewState() *State {
	return &State{
		TrueFacts:      make(map[string][]*core.Predicate),
		FalseFacts:     make(map[string][]*core.Predicate),
		AllAtomics:     bitset.NewIntSet(),
		atomicsByName:  make(map[string]*core.Atomic),
		atomicsByIndex: make(map[uint]*core.Atomic),
		Types:          make(map[string]*core.Type),
		numericAtomics: make([]*core.Atomic, 1000000),
	}
}

func (s *State) Answer(p *core.Predicate, frame *core.Frame, ctx core.QueryContext) <-chan *core.Predicate {
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
				case <-ctx.Done():
					goto done
				default:
					// continue
				}
			}
		}
		// check for types
		if pType, ok := s.Types[p.Definition.Functor]; ok && pType != nil {
			// this must have a single argument, and either be a variable or atomic
			if len(p.VarRefs) != 1 {
				panic("Type predicate has multiple arguments: " + p.String())
			}
			arg := p.VarRefs[0].Dereference()
			if arg.Ref == nil {
				// if this has a variable argument, generate all valid atomics for this type
				for ok, aIndex := pType.Atomics.GetFirstValue(); ok; ok, aIndex = pType.Atomics.GetNextValue(aIndex) {
					np := p.Clone().(*core.Predicate)
					np.VarRefs[0].Ref = s.atomicsByIndex[aIndex]
					answers <- np
					select {
					case <-ctx.Done():
						goto done
					default:
						// continue
					}
				}
				answers <- core.Terminate
				goto done
			}
			if _, ok := arg.Ref.(*core.Atomic); ok {
				answers <- p
				answers <- core.Terminate
				goto done
			}
		}
	done:
		close(answers)
	}()
	return answers
}

func (s *State) GetName() string {
	return "State"
}

func (s *State) GetType(name string) *core.Type {
	if t := s.Types[name]; t != nil {
		return t
	}
	if name == "Numeric" {
		return Numeric
	}
	t := &core.Type{
		Name:    name,
		Atomics: bitset.NewIntSet(),
	}
	s.Types[name] = t
	return t
}

func (s *State) GetNumericAtomic(index uint) *core.Atomic {
	if s.numericAtomics[index] == nil || index >= uint(len(s.numericAtomics)) {
		a := &core.Atomic{
			Index: index,
			Value: strconv.Itoa(int(index)),
			Type:  Numeric,
		}
		if index < uint(len(s.numericAtomics)) {
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
		return s.GetNumericAtomic(uint(atomicIndex))
	}
	//handle non-numeric atomics
	atomicIndex := uint(0)
	if t != nil {
		// try the next index for this type
		var ok bool
		ok, atomicIndex = t.Atomics.GetLastValue()
		atomicIndex++
		// now check that it is unused in all atomics
		if !ok || s.AllAtomics.Contains(atomicIndex) {
			// not free. So use an new global atomic index
			_, atomicIndex = s.AllAtomics.GetLastValue()
			atomicIndex += 5
		}
	} else {
		_, atomicIndex = s.AllAtomics.GetLastValue()
		atomicIndex += 1
	}
	atomic := &core.Atomic{
		Index: atomicIndex,
		Value: name,
		Type:  t,
	}
	s.AllAtomics.Add(atomicIndex)
	s.atomicsByName[name] = atomic
	s.atomicsByIndex[atomicIndex] = atomic
	if t != nil {
		t.Atomics.Add(atomicIndex)
	}
	return atomic
}

func (s *State) SetTrue(p *core.Predicate) {
	if !p.IsFact() {
		panic("predicate is not a fact: " + p.String())
	}
	if s.TrueFacts[p.Definition.Functor] == nil {
		s.TrueFacts[p.Definition.Functor] = make([]*core.Predicate, 0)
	}
	//TODO: just return if the predicate is already in the list
	s.TrueFacts[p.Definition.Functor] = append(s.TrueFacts[p.Definition.Functor], p)
	//TODO: remove from false facts, if it exists
}

func (s *State) SetFalse(p *core.Predicate) {
	if !p.IsFact() {
		panic("predicate is not a fact: " + p.String())
	}
	if s.FalseFacts[p.Definition.Functor] == nil {
		s.FalseFacts[p.Definition.Functor] = make([]*core.Predicate, 0)
	}
	s.FalseFacts[p.Definition.Functor] = append(s.FalseFacts[p.Definition.Functor], p)
	//TODO: remove from true facts, if it exists
}
