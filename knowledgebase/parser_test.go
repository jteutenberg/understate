package knowledgebase_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/jteutenberg/bitset-go"
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
			{Label: "A", Type: nil},
			{Label: "B", Type: nil},
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
		t.Errorf("expected variable X, got %v", predicate.(*core.Predicate).VarRefs[0].Dereference())
	}
	if predicate.(*core.Predicate).VarRefs[1].Label != "Y" {
		t.Errorf("expected variable Y, got %v", predicate.(*core.Predicate).VarRefs[1].Dereference())
	}
}

func TestPreparedExamples(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	person := &core.Type{
		Name:    "Person",
		Atomics: bitset.NewIntSet(),
	}
	//kb.AddAtomic("sam", person)
	//kb.AddAtomic("alex", person)
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "parent",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "Parent", Type: person},
			{Label: "Child", Type: person},
		},
	})

	file, err := os.Open("../tests/input1.txt")
	if err != nil {
		t.Fatalf("failed to open test input file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		lineNum++
		parsed, _, err := kb.ParseClause(line, nil)
		if err != nil {
			t.Errorf("error parsing line %d (%q): %v", lineNum, line, err)
			continue
		}
		if parsed == nil {
			t.Errorf("line %d: parser returned nil for line: %q", lineNum, line)
		}
		if p, ok := parsed.(*core.Predicate); ok {
			if p.IsFact() {
				kb.SetTrue(p)
			} else {
				answers := kb.Answer(p)
				for ans := range answers {
					if ans == nil {
						t.Logf("%s: Done.", p.String())
						break
					}
					t.Logf("%s -> %v", p.String(), ans)
				}
			}
		}
	}

}
