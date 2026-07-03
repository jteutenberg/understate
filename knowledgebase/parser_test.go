package knowledgebase_test

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/calculator"
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/io"
	"github.com/jteutenberg/understate/knowledgebase"
	"github.com/jteutenberg/understate/rules"
	"github.com/jteutenberg/understate/state"
)

func TestParseArguments(t *testing.T) {
	kb := knowledgebase.NewKnowledgeBase()
	frame := core.NewFrame()
	args, err := kb.ParseArguments("X, Y, Z", nil, frame)
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
	frame := core.NewFrame()
	atomic, next, err := kb.ParseClause("cow", nil, frame)
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
	frame := core.NewFrame()
	atomic, next, err := kb.ParseClause("cow, dog", nil, frame)
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
	frame := core.NewFrame()
	variable, next, err := kb.ParseClause("Xenon", nil, frame)
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
	frame := core.NewFrame()
	predicate, next, err := kb.ParseClause("eat(X, Y)", nil, frame)
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

func relationsKnowledgeBase() (*knowledgebase.KnowledgeBase, *rules.RuleMachine) {
	kb := knowledgebase.NewKnowledgeBase()
	rules := rules.NewRuleMachine(kb, kb.State)
	kb.AddAnswerer(rules)
	kb.AddAnswerer(calculator.NewCalculator(kb.State))
	kb.AddPredicateDefinition(calculator.Gt)
	kb.AddPredicateDefinition(calculator.Sum)
	person := &core.Type{
		Name:    "Person",
		Atomics: bitset.NewIntSet(),
	}
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "parent",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "Parent", Type: person},
			{Label: "Child", Type: person},
		},
	})
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "sibling",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "A", Type: person},
			{Label: "B", Type: person},
		},
	})
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "grandparent",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "Grandparent", Type: person},
			{Label: "Grandchild", Type: person},
		},
	})
	return kb, rules
}

func linesKnowledgeBase() (*knowledgebase.KnowledgeBase, *rules.RuleMachine) {
	kb := knowledgebase.NewKnowledgeBase()
	rules := rules.NewRuleMachine(kb, kb.State)
	kb.AddAnswerer(rules)
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "point",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "X", Type: state.Numeric},
			{Label: "Y", Type: state.Numeric},
		},
	})
	kb.AddPredicateDefinition(&core.PredicateDefinition{
		Functor: "line",
		ArgDefinitions: []core.ArgumentDefinition{
			{Label: "A", Type: nil},
			{Label: "B", Type: nil},
		},
	})
	return kb, rules
}

func TestParseExamples1(t *testing.T) {
	doParseExamples("../tests/input1.txt", t)
}
func TestParseExamples2(t *testing.T) {
	doParseExamples("../tests/input2.txt", t)
	doParseExamples("../tests/input2a.txt", t)
}
func TestParseExamples3(t *testing.T) {
	doParseExamples("../tests/input3.txt", t)
}
func TestParseExamples4(t *testing.T) {
	doParseExamples("../tests/input4.txt", t)
}
func TestParseExamples5(t *testing.T) {
	doParseExamples("../tests/input5.txt", t)
}

func doParseExamples(filename string, t *testing.T) {
	kb, _ := relationsKnowledgeBase()
	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open test input file: %v", err)
	}
	defer file.Close()
	parser := io.NewPredicateReader([]byte{knowledgebase.ActionSeparator, knowledgebase.RuleSeparator},
		[]byte{knowledgebase.AssertTerminator, knowledgebase.CommandTerminator, knowledgebase.QueryTerminator})
	tokens := parser.Parse(bufio.NewReader(file))
	queries := make(chan []*core.Predicate)

	for token := range tokens {
		query, _, frame, err := kb.Process(token, queries, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		if query != nil {
			fmt.Println("Query: ")
			for _, p := range query {
				fmt.Println(" - ", p.String())
			}
			if len(query) == 1 {
				// single query
				answers := kb.Answer(query[0], frame, core.NewQueryContext())
				for ans := range answers {
					fmt.Println("  -> ", ans.String())
				}
				fmt.Println("Done.")
			} else if len(query) > 1 {
				// conjunction
				answers := core.AnswerConjunction(kb, query, frame, core.NewQueryContext())
				for ans := range answers {
					fmt.Println("  ->")
					for _, p := range ans {
						fmt.Println("    ", p.String())
					}
				}
			}
		}
	}
}
