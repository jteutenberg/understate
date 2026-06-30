package knowledgebase_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jteutenberg/bitset-go"
	"github.com/jteutenberg/understate/calculator"
	"github.com/jteutenberg/understate/core"
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

func TestPreparedExamples(t *testing.T) {
	doPreparedExamples(t, false)
}
func TestPreparedExamplesHaltEarly(t *testing.T) {
	doPreparedExamples(t, true)
}

func TestParseExamples(t *testing.T) {
	doParseExamples(t)
}

func doParseExamples(t *testing.T) {
	kb, rules := relationsKnowledgeBase()
	file, err := os.Open("../tests/input1.txt")
	if err != nil {
		t.Fatalf("failed to open test input file: %v", err)
	}
	defer file.Close()
	tokens := kb.SplitInput(bufio.NewReader(file))
	for token := range tokens {
		result := kb.Parse(token)
		if result == nil {
			continue
		}
		if result.Rule != nil {
			fmt.Println("parsed rule")
			rules.AddRule(result.Rule)
		}
		if result.Predicates != nil {
			if result.IsQuery {
				fmt.Println("Query: ")
				for _, p := range result.Predicates {
					fmt.Println(" - ", p.String())
				}
				if len(result.Predicates) == 1 {
					// single query
					answers := kb.Answer(result.Predicates[0], result.Frame, core.NewQueryContext())
					for ans := range answers {
						fmt.Println("  -> ", ans.String())
					}
					fmt.Println("Done.")
				} else if len(result.Predicates) > 1 {
					// conjunction
					answers := core.AnswerConjunction(kb, result.Predicates, result.Frame, core.NewQueryContext())
					for ans := range answers {
						fmt.Println("  ->")
						for _, p := range ans {
							fmt.Println("    ", p.String())
						}
					}
				}
			} else {
				for i, p := range result.Predicates {
					fmt.Printf("parsed fact %d: %s\n", i, p.String())
					// TODO: handle not fact
					kb.SetTrue(p)
				}
			}
		}
	}
}

func doPreparedExamples(t *testing.T, haltEarly bool) {
	kb, rules := relationsKnowledgeBase()
	//file, err := os.Open("../tests/input1.txt")
	//file, err := os.Open("../tests/input2.txt")
	//file, err := os.Open("../tests/input2a.txt")
	file, err := os.Open("../tests/input4.txt")
	//kb, rules := linesKnowledgeBase()
	//file, err := os.Open("../tests/input3.txt")
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
		if line[0] == '#' {
			continue
		}
		isQuery := line[len(line)-1] == '?'
		lineNum++
		// if line contains ':-', then parse it as a rule
		frame := core.NewFrame()
		if strings.Contains(line, ":-") {
			rule, err := kb.ParseRule(line)
			if err != nil {
				t.Errorf("error parsing rule line %d (%q): %v", lineNum, line, err)
				continue
			}
			rules.AddRule(rule)
		} else {
			parsed, _, err := kb.ParseClause(line, nil, frame)
			if err != nil {
				t.Errorf("error parsing line %d (%q): %v", lineNum, line, err)
				continue
			}
			if parsed == nil {
				t.Errorf("line %d: parser returned nil for line: %q", lineNum, line)
				continue
			}
			if p, ok := parsed.(*core.Predicate); ok {
				fmt.Printf("parsed: %s is fact: %v\n", p.String(), p.IsFact())
				if p.IsFact() && !isQuery {
					kb.SetTrue(p)
				} else {
					doHalt := haltEarly
					ctx := core.NewQueryContext()
					defer ctx.Cancel()
					answers := kb.Answer(p, frame, ctx)
					answered := false
					for ans := range answers {
						if ans == nil || ans == core.Terminate {
							if p.IsFact() {
								if answered {
									t.Logf("%s: True.", p.String())
								} else {
									t.Logf("%s: False.", p.String())
								}
							} else {
								t.Logf("%s: Done.", p.String())
							}
							break
						}
						answered = true
						if !p.IsFact() {
							fmt.Printf("%s -> %v %v\n", p.String(), ans, p.CanUnify(ans))
							f2 := frame.Clone()
							p2 := p.CloneInFrame(f2)
							uerr := p2.Unify(ans)
							if uerr != nil {
								fmt.Println("Error unifying: ", uerr)
							}
						}
						if doHalt {
							fmt.Println("Halting early")
							ctx.Cancel()
							doHalt = false
						}
					}
				}
			}
		}
	}

}
