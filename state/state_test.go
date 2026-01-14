package state

import (
	"testing"

	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/core"
)

func setupTest() (map[string]*core.Type, map[string]*core.PredicateDefinition, map[string]*core.Atomic) {
	cow := core.Atomic{Index: 1, Value: "cow"}
	dog := core.Atomic{Index: 2, Value: "dog"}
	grass := core.Atomic{Index: 3, Value: "grass"}
	apple := core.Atomic{Index: 4, Value: "apple"}
	beef := core.Atomic{Index: 5, Value: "beef"}

	atomics := make(map[string]*core.Atomic)
	atomics["cow"] = &cow
	atomics["dog"] = &dog
	atomics["grass"] = &grass
	atomics["apple"] = &apple
	atomics["beef"] = &beef
	types := make(map[string]*core.Type)
	predDefs := make(map[string]*core.PredicateDefinition)
	types["Plant"] = &core.Type{
		Name:    "Plant",
		Atomics: bitset.NewIntSetFromInts([]int{grass.Index, apple.Index}),
	}
	types["Food"] = &core.Type{
		Name:    "Food",
		Atomics: bitset.NewIntSetFromInts([]int{grass.Index, apple.Index, beef.Index}),
	}
	types["Creature"] = &core.Type{
		Name:    "Creature",
		Atomics: bitset.NewIntSetFromInts([]int{cow.Index, dog.Index}),
	}
	types["Everything"] = &core.Type{
		Name:    "Everything",
		Atomics: bitset.NewIntSetFromInts([]int{cow.Index, dog.Index, grass.Index, apple.Index, beef.Index}),
	}
	// true fact: cow eats grass
	predDefs["eat"] = &core.PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "Eater", Type: types["Creature"]},
			{Label: "Food", Type: types["Food"]},
		},
	}

	return types, predDefs, atomics
}

func basicState(defs map[string]*core.PredicateDefinition, atomics map[string]*core.Atomic) *State {
	trueFact := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: atomics["cow"]},
			{Label: "Y", Ref: atomics["grass"]},
		},
	}
	trueFact2 := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: atomics["dog"]},
			{Label: "Y", Ref: atomics["beef"]},
		},
	}
	trueFact3 := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: atomics["cow"]},
			{Label: "Y", Ref: atomics["apple"]},
		},
	}
	// false fact: cow does not eat beef
	falseFact := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: atomics["cow"]},
			{Label: "Y", Ref: atomics["beef"]},
		},
	}
	return &State{
		TrueFacts: map[string][]*core.Predicate{
			"eat": {trueFact, trueFact2, trueFact3},
		},
		FalseFacts: map[string][]*core.Predicate{
			"eat": {falseFact},
		},
	}

}

func TestStateAnswerQuery(t *testing.T) {
	_, defs, atomics := setupTest()
	state := basicState(defs, atomics)

	//find what cows eat
	query := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: atomics["cow"]},
			{Label: "Y", Ref: nil},
		},
	}
	answer, _ := state.Answer(query)
	ansCount := 0
	for ans := range answer {
		ansCount++
		//any order is valid
		if !((ans[0] == atomics["cow"] && ans[1] == atomics["grass"]) ||
			(ans[0] == atomics["cow"] && ans[1] == atomics["apple"])) {
			t.Errorf("unexpected answer %v eats %v", ans[0], ans[1])
		}
	}
	if ansCount != 2 {
		t.Errorf("expected 2 answers, got %d", ansCount)
	}
}

func TestStateAnswerQueryPermutations(t *testing.T) {
	_, defs, atomics := setupTest()
	state := basicState(defs, atomics)

	//find what cows eat
	query := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: nil},
			{Label: "Y", Ref: nil},
		},
	}
	answer, _ := state.Answer(query)
	ansCount := 0
	for ans := range answer {
		ansCount++
		//any order is valid
		if !((ans[0] == atomics["cow"] && ans[1] == atomics["grass"]) ||
			(ans[0] == atomics["dog"] && ans[1] == atomics["beef"]) ||
			(ans[0] == atomics["cow"] && ans[1] == atomics["apple"])) {
			t.Errorf("unexpected answer %v eats %v", ans[0], ans[1])
		}
	}
	if ansCount != 3 {
		t.Errorf("expected 3 answers, got %d", ansCount)
	}
}
func TestStateAnswerQueryFact(t *testing.T) {
	_, defs, atomics := setupTest()
	state := basicState(defs, atomics)

	//find what cows eat
	query := &core.Predicate{
		Definition: defs["eat"],
		VarLabels:  []string{"?1", "?2"},
		VarRefs: []*core.VariableReference{
			{Label: "X", Ref: atomics["cow"]},
			{Label: "Y", Ref: atomics["grass"]},
		},
	}
	answer, _ := state.Answer(query)
	ansCount := 0
	for ans := range answer {
		ansCount++
		//any order is valid
		if !(ans[0] == atomics["cow"] && ans[1] == atomics["grass"]) {
			t.Errorf("unexpected answer %v eats %v", ans[0], ans[1])
		}
	}
	if ansCount != 1 {
		t.Errorf("expected 1 answer, got %d", ansCount)
	}
}
