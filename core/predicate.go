package core

import (
	"fmt"
)

var Terminate = &Predicate{Definition: &PredicateDefinition{Functor: "Terminate"}}
var Pass = &Predicate{Definition: &PredicateDefinition{Functor: "Pass"}}

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
	VarRefs    []*VariableReference
}

type Answerer interface {
	Answer(p *Predicate, halt <-chan bool) <-chan *Predicate
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

func (p *Predicate) GetArgument(index int) Unifiable {
	return p.VarRefs[index].Dereference().Ref
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
	if p, ok := r.Ref.(*Predicate); ok {
		return p.String()
	}
	return "BadRef"
}

func (r *VariableReference) StringVerbose() string {
	if v, ok := r.Ref.(*VariableReference); ok {
		return r.Label + " -> " + v.StringVerbose()
	}
	return "[" + r.Label + "]" + r.String()
}

func (r *VariableReference) Dereference() *VariableReference {
	for ref, ok := r.Ref.(*VariableReference); ok; ref, ok = r.Ref.(*VariableReference) {
		r = ref
	}
	return r
}

func (r *VariableReference) Unify(other Unifiable) error {
	// check for final referencing two non-variables. Call their Unify.
	finalR := r.Dereference()
	finalOtherRef := other
	if vRefOther, ok := other.(*VariableReference); ok {
		finalOther := vRefOther.Dereference()
		finalOtherRef = finalOther.Ref
		// check for same variable reference
		if finalR == finalOther {
			return nil
		}
		// if this is a variable too, then just point to the other
		if finalR.Ref == nil {
			finalR.Ref = finalOther
			return nil
		} else if finalOtherRef == nil {
			return finalOther.Unify(finalR.Ref)
		}
	}
	// we now have sorted the cases for
	// - both were the same variable
	// - both were different variables
	// - the other was a variable reference to something concrete

	// now if this is variable but other is directly concrete
	if finalR.Ref == nil {
		finalR.Ref = finalOtherRef
		return nil
	}

	// and finally if both are non-variables, unify them
	return finalR.Ref.Unify(finalOtherRef)
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

func (a *Predicate) CloneWithVars(vars map[string]*VariableReference) *Predicate {
	p := &Predicate{
		Definition: a.Definition,
		VarRefs:    make([]*VariableReference, len(a.VarRefs)),
	}
	for i, varRef := range a.VarRefs {
		// handle recursive predicates
		d := varRef.Dereference()
		if p2, ok := d.Ref.(*Predicate); ok {
			// find the variable references for this predicate too
			p2 = p2.CloneWithVars(vars)
			// and make a new reference to it, not shared.
			p.VarRefs[i] = &VariableReference{
				Label: varRef.Label,
				Ref:   p2,
			}
			continue
		}
		p.VarRefs[i] = vars[varRef.Label]
	}
	return p
}

func (a *Predicate) Clone() Unifiable {
	p := &Predicate{
		Definition: a.Definition,
		VarRefs:    make([]*VariableReference, len(a.VarRefs)),
	}
	newVars := make(map[string]*VariableReference)
	needPredicates := make([]*VariableReference, 0, len(a.VarRefs))
	origPredicates := make([]*Predicate, 0, len(a.VarRefs))
	// clone everything except predicates
	for i, varRef := range a.VarRefs {
		v := varRef.Dereference()
		if _, ok := newVars[v.Label]; !ok {
			if op, pok := v.Ref.(*Predicate); pok {
				vr := &VariableReference{
					Label: v.Label,
					Ref:   nil,
				}
				newVars[v.Label] = vr
				needPredicates = append(needPredicates, vr)
				origPredicates = append(origPredicates, op)
			} else {
				vr := &VariableReference{
					Label: v.Label,
					Ref:   nil,
				}
				if v.Ref != nil {
					vr.Ref = v.Ref.Clone()
				}
				newVars[v.Label] = vr
			}
		}
		p.VarRefs[i] = newVars[v.Label]
	}
	// clone the predicates
	for i, vr := range needPredicates {
		vr.Ref = origPredicates[i].CloneWithVars(newVars)
	}
	return p
}
