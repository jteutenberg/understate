package knowledgebase

import (
	"fmt"

	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/io"
	"github.com/jteutenberg/understate/rules"
	"github.com/jteutenberg/understate/state"
)

// constants for terminators and separators in text input
const (
	AssertTerminator  = '.'
	QueryTerminator   = '?'
	CommandTerminator = '!'
	RuleSeparator     = '~' // between lhs and rhs
	ActionSeparator   = '|' // between preconditions and effects
)

type ParsedPredicates struct {
	Predicates [][]*core.Predicate
	IsQuery    bool
	IsCommand  bool
}

func (kb *KnowledgeBase) Process(input io.ParseResult, queriesOut chan<- []*core.Predicate, actionsOut chan<- []*core.Predicate) (query []*core.Predicate, action *core.Predicate, frame *core.Frame, err error) {
	frame = core.NewFrame()
	// check for a definition override (for predicate definitions only)
	isDefinition := input.Predicates[0][0] == ':'
	if isDefinition {
		if input.Terminator != AssertTerminator {
			return nil, nil, nil, fmt.Errorf("Error. Definitions should terminate as a fact")
		}
		// Definitions!
		input.Predicates[0] = input.Predicates[0][1:]
		if len(input.Separators) > 0 {
			return nil, nil, nil, fmt.Errorf("Error. Definitions should not include separators")
		}
		// add predicate definitions
		for _, s := range input.Predicates {
			if pdef, _, err := kb.ParseDefinition(s); err == nil {
				// A predicate definition is a functor and list of types, e.g.
				// :eat(Herbivore, Plant)
				kb.AddPredicateDefinition(pdef)
			} else {
				return nil, nil, nil, fmt.Errorf("Error parsing definition: %v", err)
			}
		}
		return nil, nil, nil, nil
	}

	// everything else is a list of predicate sets
	ps := make([][]*core.Predicate, len(input.Predicates))
	for i, s := range input.Predicates {
		if sps, err := kb.ParsePredicates(s, frame); err == nil {
			ps[i] = sps
		} else {
			return nil, nil, nil, fmt.Errorf("Error. Unable to parse predicates: %v", err)
		}
	}
	// check for a rule (definition)
	if len(input.Separators) >= 1 && input.Separators[0] == RuleSeparator {
		if len(ps) != 2 {
			return nil, nil, nil, fmt.Errorf("Error. Rule should have a LHS and RHS set of predicates")
		}
		if len(ps[0]) != 1 {
			return nil, nil, nil, fmt.Errorf("Error. LHS of rule must have only one predicate.")
		}
		rule := rules.NewRule(ps[0][0], ps[1], frame)
		// add the rule to the knowledge base's tule machine
		for _, answerer := range kb.answerers {
			if ruler, ok := answerer.(*rules.RuleMachine); ok {
				ruler.AddRule(rule)
				return nil, nil, nil, nil
			}
		}
		return nil, nil, nil, fmt.Errorf("Error. No rule machine found to add rule")
	} else if input.Terminator == QueryTerminator {
		if len(input.Separators) > 0 {
			return nil, nil, nil, fmt.Errorf("Error. Queries should not include separators")
		}
		// we now get list of predicates (potentially an action's predicate)
		//TODO: check for an action name?
		return ps[0], nil, frame, nil
	} else if input.Terminator == AssertTerminator {
		if len(input.Separators) > 0 {
			return nil, nil, nil, fmt.Errorf("Error. Assertions should not include separators")
		}
		// typically a single ground fact
		for _, nps := range ps {
			for _, p := range nps {
				kb.SetTrue(p)
			}
		}
		return nil, nil, nil, nil
	} else if input.Terminator == CommandTerminator {
		// an action
		// looks like a predicate but won't have a definition
	}
	return nil, nil, nil, fmt.Errorf("Unkown input type")
}

func (kb *KnowledgeBase) ParseArguments(s string, typeHints []*core.Type, frame *core.Frame) ([]core.Unifiable, error) {
	args := make([]core.Unifiable, 0, 5)
	nextTypeHint := 0
	for i := 0; i < len(s); {
		var typeHint *core.Type
		if nextTypeHint < len(typeHints) {
			typeHint = typeHints[nextTypeHint]
			nextTypeHint++
		}
		c, next, err := kb.ParseClause(s[i:], typeHint, frame)
		if err != nil {
			return nil, err
		}
		args = append(args, c)
		if next < 1 {
			return nil, fmt.Errorf("invalid clause: %s got index %d", s[i:], next)
		}
		i += next
	}
	return args, nil
}

func (kb *KnowledgeBase) ParsePredicate(functor, arguments string, frame *core.Frame) (*core.Predicate, error) {
	pdef, ok := kb.predicateDefinitions[functor]
	if !ok {
		//TODO: create a new predicate definition, nil typeHints
		return nil, fmt.Errorf("unknown predicate: %s", functor)
	}
	//j is one after the close parenthesis, args end two char earlier
	typeHints := make([]*core.Type, len(pdef.ArgDefinitions))
	for i, argDef := range pdef.ArgDefinitions {
		typeHints[i] = argDef.Type
	}
	args, err := kb.ParseArguments(arguments, typeHints, frame)
	if err != nil {
		return nil, err
	}
	if len(args) != len(pdef.ArgDefinitions) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(pdef.ArgDefinitions), len(args))
	}
	// TODO: if any argDef has nil type in a new predicate definition, then
	// try inferring from the arguments, e.g. atomic types or predicate definitions
	labels := make([]string, len(args))
	for i, argDef := range pdef.ArgDefinitions {
		if atomic, ok := args[i].(*core.Atomic); ok {
			if argDef.Type == nil {
				argDef.Type = atomic.Type
			}
		}
		labels[i] = argDef.Label
	}
	return core.NewPredicate(pdef, labels, args, frame), nil
}

func (kb *KnowledgeBase) ParseDefinitionArguments(s string, parent *core.PredicateDefinition) error {
	for i := 0; i < len(s); i++ {
		fmt.Println("Parsing definition arguments: ", s)
		// probe for an atomic argument type: when there is a comma or close parenthesis and no open parenthesis
		split := i
		for j := i; j < len(s); j++ {
			if s[j] == ':' {
				split = j
				continue
			}
			if s[j] == ',' || s[j] == ')' || j == len(s)-1 {
				if j == len(s)-1 {
					j += 1
				}
				// argument definition is give as Label:Type
				label := s[i:split]
				typeName := s[split+1 : j]
				t := kb.State.GetType(typeName)
				parent.ArgDefinitions = append(parent.ArgDefinitions, core.ArgumentDefinition{
					Label: label,
					Type:  t,
				})
				i = j // this will be incremented by the loop
				fmt.Println("Parsed definition argument: ", label, " with type ", t.Name)
				break
			}
			if s[j] == '(' {
				label := s[i:split]
				// this is a nested predicate definition
				subDef, n, err := kb.ParseDefinition(s[split+1 : j-1])
				if err != nil {
					return err
				}
				parent.ArgDefinitions = append(parent.ArgDefinitions, core.ArgumentDefinition{
					Label:         label,
					SubDefinition: subDef,
				})
				i += n + split + 1
				break
			}
		}
	}
	return nil
}

func (kb *KnowledgeBase) ParseDefinition(s string) (*core.PredicateDefinition, int, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == '(' {
			// this will be a predicate. Parse its arguments
			functor := s[:i]
			j := i + 1
			count := 1
			for count > 0 && j < len(s) {
				if s[j] == '(' {
					count++
				} else if s[j] == ')' {
					count--
				}
				j++
			}
			if count != 0 {
				return nil, 0, fmt.Errorf("missing close parenthesis")
			}
			// each argument is either a type or another predicate definition
			def := core.PredicateDefinition{
				Functor:        functor,
				ArgDefinitions: make([]core.ArgumentDefinition, 0, 5),
			}
			err := kb.ParseDefinitionArguments(s[i+1:j-1], &def)
			if err != nil {
				return nil, 0, err
			}
			return &def, j, nil
		}
	}
	return nil, 0, fmt.Errorf("invalid definition: %s", s)
}

func (kb *KnowledgeBase) ParseClause(s string, typeHint *core.Type, frame *core.Frame) (core.Unifiable, int, error) {
	// consume the lead string up to the first '('',', ')', or end of string
	for i := 0; i < len(s); i++ {
		if s[i] == '(' {
			// this will be a predicate. Parse its arguments
			functor := s[:i]
			j := i + 1
			count := 1
			for count > 0 && j < len(s) {
				if s[j] == '(' {
					count++
				} else if s[j] == ')' {
					count--
				}
				j++
			}
			if count != 0 {
				return nil, 0, fmt.Errorf("missing close parenthesis")
			}
			predicate, err := kb.ParsePredicate(functor, s[i+1:j-1], frame)
			if err != nil {
				return nil, 0, err
			}
			// skip any following comma
			for j < len(s) && (s[j] == ' ' || s[j] == ',') {
				j++
			}
			return predicate, j, nil
		} else if i == len(s)-1 || s[i] == ',' || s[i] == ')' {
			// this is an atomic or variable, possibly with a terminating char
			if i == len(s)-1 {
				i++
			}
			// skip any following comma
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == ',') {
				j++
			}
			if s[0] >= 'a' && s[0] <= 'z' {
				// atomic
				return kb.State.GetAtomic(s[:i], typeHint), j, nil
			} else if s[0] >= '1' && s[0] <= '9' {
				// numeric
				return kb.State.GetAtomic(s[:i], state.Numeric), j, nil
			} else {
				label := s[:i]
				if frame.Vars[label] == nil {
					frame.Vars[label] = &core.VariableReference{
						Label: label,
						Ref:   nil,
					}
				}
				return frame.Vars[label], j, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("invalid clause: %s", s)
}

func (kb *KnowledgeBase) ParsePredicates(s string, frame *core.Frame) ([]*core.Predicate, error) {
	ps := make([]*core.Predicate, 0, 5)
	args, err := kb.ParseArguments(s, nil, frame)
	if err != nil {
		return nil, err
	}
	for _, arg := range args {
		if predicate, ok := arg.(*core.Predicate); ok {
			ps = append(ps, predicate)
		} else {
			return nil, fmt.Errorf("Expected predicate, got %T", arg)
		}
	}
	return ps, nil
}

/*
func (kb *KnowledgeBase) ParseRule(s string) (*rules.Rule, error) {
	frame := core.NewFrame()
	// consume the lead string up to the first ':-'
	for i := 0; i < len(s)-1; i++ {
		if s[i] == ':' && s[i+1] == '-' {
			// this is a rule. Parse its lhs and rhs
			var lhs *core.Predicate
			if lhsClause, _, err := kb.ParseClause(s[:i], nil, frame); err != nil {
				return nil, err
			} else {
				lhs = lhsClause.(*core.Predicate)
			}
			i += 2
			for i < len(s) && s[i] == ' ' {
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("expected rule RHS, got %q", s[i:])
			}
			end := i + 1
			for end < len(s) && s[end] != '.' {
				end++
			}
			// then parse multiple comma delimited predicates
			rhs := make([]*core.Predicate, 0, 5)
			args, err := kb.ParseArguments(s[i:end], nil, frame)
			if err != nil {
				return nil, err
			}
			for i, arg := range args {
				if predicate, ok := arg.(*core.Predicate); ok {
					rhs = append(rhs, predicate)
				} else {
					return nil, fmt.Errorf("Expected predicate in rule's number %d RHS, got %T", i, arg)
				}
			}
			return rules.NewRule(lhs, rhs, frame), nil
		}
	}
	return nil, fmt.Errorf("invalid rule: %s", s)
}*/
