package knowledgebase_test

import (
	"testing"

	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/knowledgebase"
)

func TestParseArguments(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	args, err := kb.ParseArguments("X, Y, Z", nil)
	// three variables
	if err != nil {
		t.Errorf("error parsing arguments: %v", err)
	}
	if len(args) != 3 {
		t.Errorf("expected 3 arguments, got %d", len(args))
	}
	if args[0].(*core.VariableReference).Label != "X" {
		t.Errorf("expected argument 0 to be X, got %v", args[0])
	}
}

func TestParseClauseAtomic(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	atomic, next, err := kb.ParseClause("cow", nil)
	if err != nil {
		t.Errorf("error parsing atomic: %v", err)
	}
	if next < 3 {
		t.Errorf("expected next to be >= 3, got %d", next)
	}
	if atomic.(*core.Atomic).Value != "cow" {
		t.Errorf("expected atomic to be cow, got %v", atomic)
	}
}

func TestParseClauseAtomicWithTrailingChar(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	atomic, next, err := kb.ParseClause("cow, dog", nil)
	if err != nil {
		t.Errorf("error parsing atomic: %v", err)
	}
	if next != 5 {
		t.Errorf("expected next to be 5, got %d", next)
	}
	if atomic.(*core.Atomic).Value != "cow" {
		t.Errorf("expected atomic to be cow, got %v", atomic)
	}
}

func TestParseClauseVariable(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	variable, next, err := kb.ParseClause("Xenon", nil)
	if err != nil {
		t.Errorf("error parsing variable: %v", err)
	}
	if next < 5 {
		t.Errorf("expected next to be >= 5, got %d", next)
	}
	if variable.(*core.VariableReference).Label != "Xenon" {
		t.Errorf("expected variable to be Xenon, got %v", variable)
	}
}

func TestParseClausePredicate(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "eat",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "X", Type: nil},
			{Label: "Y", Type: nil},
		},
	})
	predicate, next, err := kb.ParseClause("eat(X, Y)", nil)
	if err != nil {
		t.Errorf("error parsing predicate: %v", err)
	}
	if next < 9 {
		t.Errorf("expected next to be >= 9, got %d", next)
	}
	if predicate.(*core.Predicate).Definition.Functor != "eat" {
		t.Errorf("expected predicate to be eat, got %v", predicate)
	}
	if predicate.(*core.Predicate).VarRefs[0].Label != "X" {
		t.Errorf("expected variable X, got %v", predicate.(*core.Predicate).VarRefs[0])
	}
	if predicate.(*core.Predicate).VarRefs[1].Label != "Y" {
		t.Errorf("expected variable Y, got %v", predicate.(*core.Predicate).VarRefs[1])
	}
}
