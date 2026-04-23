package core

import (
	"testing"

	"github.com/jteutenberg/bitset-go"
)

func TestSimpleUnify(t *testing.T) {
	cow := Atomic{Index: 1, Value: "cow"}
	grass := Atomic{Index: 2, Value: "grass"}
	plantType := Type{
		Name:    "Plant",
		Atomics: bitset.NewIntSetFromInts([]int{grass.Index}),
	}
	herbivoreType := Type{
		Name:    "Herbivore",
		Atomics: bitset.NewIntSetFromInts([]int{cow.Index}),
	}
	def := &PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []ArgumentDefinition{
			{Label: "Eater", Type: &herbivoreType},
			{Label: "Food", Type: &plantType},
		},
	}

	predA := &Predicate{
		Definition: def,
		VarRefs: []*VariableReference{
			{Label: "X", Ref: &cow},
			{Label: "Y", Ref: nil},
		},
	}
	predB := &Predicate{
		Definition: def,
		VarRefs: []*VariableReference{
			{Label: "A", Ref: nil},
			{Label: "B", Ref: &grass},
		},
	}
	if !predA.CanUnify(predB) {
		t.Errorf("predicates cannot unify")
	}
	err := predA.Unify(predB)
	if err != nil {
		t.Errorf("error unifying predicates: %v", err)
	}
	if predA.String() != "eat(cow, grass)" {
		t.Errorf("unified predicate: %v", predA)
	}
}

func TestCloneVars(t *testing.T) {
	def := &PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []ArgumentDefinition{
			{Label: "Eater", Type: nil},
			{Label: "Food", Type: nil},
		},
	}
	pred := &Predicate{
		Definition: def,
		VarRefs: []*VariableReference{
			{Label: "X", Ref: nil},
			{Label: "Y", Ref: nil},
		},
	}
	pred.VarRefs[1].Ref = pred.VarRefs[0]
	cloned := pred.Clone().(*Predicate)
	if cloned.VarRefs[1].Dereference() != cloned.VarRefs[0].Dereference() {
		t.Errorf("cloning did not preserve shared variable")
	}
	if cloned.String() != "eat(X, X)" {
		t.Errorf("cloned predicate: %v", cloned)
	}
}

func TestCloneAtomic(t *testing.T) {
	def := &PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []ArgumentDefinition{
			{Label: "Eater", Type: nil},
			{Label: "Food", Type: nil},
		},
	}
	rat := &Atomic{Index: 1, Value: "rat"}
	pred := &Predicate{
		Definition: def,
		VarRefs: []*VariableReference{
			{Label: "X", Ref: rat},
			{Label: "Y", Ref: nil},
		},
	}
	pred.VarRefs[1].Ref = pred.VarRefs[0]
	cloned := pred.Clone().(*Predicate)
	if cloned.VarRefs[1].Dereference() != cloned.VarRefs[0].Dereference() {
		t.Errorf("cloning did not preserve structure")
	}
	if cloned.String() != "eat(rat, rat)" {
		t.Errorf("cloned predicate: %v", cloned)
	}
}
