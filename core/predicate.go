package core

import (
	"fmt"

	"github.com/jteutenberg/bitset-go"
)

type ArgumentDefinition struct {
	Label         string
	Type          *Type
	SubDefinition *PredicateDefinition
}

type PredicateDefinition struct {
	Functor        string
	ArgDefinitions []ArgumentDefinition
}

type Unifiable interface {
	CanUnify(other Unifiable) bool
	Unify(other Unifiable) error
	Clone() Unifiable
}

type VariableReference struct {
	Label string    // names like ?1 ?2
	Ref   Unifiable // *Atomic, *Predicate, *VariableReference, or nil
}

type Predicate struct {
	Definition *PredicateDefinition
	VarLabels  []string // names like X, A, B, etc.
	VarRefs    []*VariableReference
}

type Answerer interface {
	Answer(p *Predicate) (<-chan []*Atomic, []*bitset.IntSet)
}

func (p *Predicate) String() string {
	s := fmt.Sprintf("%s(", p.Definition.Functor)
	for i, varRef := range p.VarRefs {
		s += varRef.String()
		if i < len(p.VarRefs)-1 {
			s += ", "
		}
	}
	s += ")"
	return s
}

func (p *Predicate) IsFact() bool {
	for _, varRef := range p.VarRefs {
		vr := varRef.Dereference()
		if vr.Ref == nil {
			return false
		} else if p2, ok := vr.Ref.(*Predicate); ok {
			if !p2.IsFact() {
				return false
			}
		}
	}
	return true
}

func (r *VariableReference) String() string {
	if r.Ref == nil {
		return r.Label
	}
	if a, ok := r.Ref.(*Atomic); ok {
		return a.String()
	}
	if v, ok := r.Ref.(*VariableReference); ok {
		return v.String()
	}
	return "BadRef"
}

func (r *VariableReference) Dereference() *VariableReference {
	for ref, ok := r.Ref.(*VariableReference); ok; ref, ok = r.Ref.(*VariableReference) {
		r = ref
	}
	return r
}

func (r *VariableReference) Unify(other Unifiable) error {
	vr := r.Dereference()
	if vr.Ref == nil {
		// point directly to the other
		r.Ref = other
		return nil
	}
	r.Ref = vr.Ref
	if other, ok := other.(*VariableReference); ok {
		other = other.Dereference()
		if other.Ref == nil {
			return nil
		}
		return vr.Ref.Unify(other.Ref)
	}
	// defer unification to a non-variable
	return r.Ref.Unify(other)
}

func (r *VariableReference) CanUnify(other Unifiable) bool {
	vr := r.Dereference()
	if vr.Ref == nil {
		return true
	}
	if bother, ok := other.(*VariableReference); ok {
		bother = bother.Dereference()
		if bother.Ref == nil {
			return true
		}
		return vr.Ref.CanUnify(bother.Ref)
	}
	return vr.Ref.CanUnify(other)
}

func (r *VariableReference) Clone() Unifiable {
	if r.Ref == nil {
		return &VariableReference{
			Label: r.Label,
			Ref:   nil,
		}
	}
	return &VariableReference{
		Label: r.Label,
		Ref:   r.Ref.Clone(),
	}
}

func (a *Predicate) Unify(other Unifiable) error {
	b, ok := other.(*Predicate)
	if !ok || a.Definition.Functor != b.Definition.Functor {
		return fmt.Errorf("functors do not match")
	}
	for i, argA := range a.VarRefs {
		argB := b.VarRefs[i]
		if err := argA.Unify(argB); err != nil {
			return err
		}
	}
	return nil
}

/*
* Unification of two Predicates. This requires matching functors and
* arguments that unify with one another.
 */
func (a *Predicate) CanUnify(other Unifiable) bool {
	b, ok := other.(*Predicate)
	if !ok {
		return false
	}
	if a.Definition.Functor != b.Definition.Functor {
		return false
	}

	// now unify each argument
	for i, argA := range a.VarRefs {
		argB := b.VarRefs[i]
		if !argA.CanUnify(argB) {
			return false
		}
	}
	return true
}

func (a *Predicate) Clone() Unifiable {
	p := &Predicate{
		Definition: a.Definition,
		VarLabels:  make([]string, len(a.VarLabels)),
		VarRefs:    make([]*VariableReference, len(a.VarRefs)),
	}
	newVars := make(map[string]*VariableReference)
	for i, varLabel := range a.VarLabels {
		p.VarLabels[i] = varLabel
		newVars[a.VarRefs[i].Label] = &VariableReference{
			Label: varLabel,
			Ref:   nil,
		}
	}
	for i, varRef := range a.VarRefs {
		newVar := newVars[varRef.Label]
		if varRef.Ref != nil {
			if existingRef, ok := varRef.Ref.(*VariableReference); ok {
				// point to the new equivalent
				newVar.Ref = newVars[existingRef.Label]
			} else {
				newVar.Ref = existingRef.Clone()
			}
			newVar.Ref = varRef.Ref.Clone()
		}
		p.VarRefs[i] = newVar
	}
	return p
}
