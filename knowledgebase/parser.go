package knowledgebase

import (
	"fmt"
	"io"
	"strings"

	"github.com/jteutenberg/understate/actions"
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/rules"
	"github.com/jteutenberg/understate/state"
)

type ParseResult struct {
	Predicates []*core.Predicate
	Frame      *core.Frame
	Rule       *rules.Rule
	Action     *actions.Action
	IsQuery    bool
}

func (kb *KnowledgeBase) SplitInput(reader io.ByteReader) <-chan string {
	result := make(chan string)
	go func() {
		line := make([]byte, 0, 10000)
		// consume until a:
		// . for a fact or definition
		// ? for a query
		// ! for an action or non-fact update
		inComment := false
		for {
			if b, err := reader.ReadByte(); err != nil {
				close(result)
				return
			} else {
				inComment = inComment || (b == '#')
				if b == ' ' || b == '\t' || b == '\n' || b == '\r' || inComment {
					// ignore whitespace
					if b == '\r' || b == '\n' {
						inComment = false
					}
					continue
				}
				line = append(line, b)
				if b == '.' || b == '?' || b == '!' {
					result <- string(line)
					line = line[:0]
				}
			}
		}
	}()
	return result
}

func (kb *KnowledgeBase) Parse(s string) *ParseResult {
	isDefinition := s[0] == ':'
	final := s[len(s)-1]
	s = s[:len(s)-1]
	if isDefinition {
		if final != '.' {
			fmt.Println("Error. Definitions should not be queries or updates")
			return nil
		}
		s = s[1:]
		fmt.Println("Parsing definition")
		// did we see a rule's ":-"?
		if strings.Contains(s, ":-") {
			if r, err := kb.ParseRule(s); err == nil {
				return &ParseResult{Rule: r, IsQuery: false}
			} else {
				fmt.Println("Error. Failed to parse rule", err)
			}
		} else if pdef, _, err := kb.ParseDefinition(s); err == nil {
			// A predicate definition is a functor and list of types, e.g.
			// :eat(Herbivore, Plant)
			kb.AddPredicateDefinition(pdef)
		} else {
			fmt.Printf("error parsing definition: %v\n", err)
		}
	} else if final == '?' {
		// we now get list of predicates (potentially an action's predicate)
		frame := core.NewFrame()
		predicates := make([]*core.Predicate, 0, 5)
		for len(s) > 0 {
			// use type hints from any defined predicates
			clause, next, err := kb.ParseClause(s, nil, frame)
			if err != nil {
				fmt.Printf("error parsing clause: %v\n", err)
				break
			}
			//TODO: check for an action name?
			if p, ok := clause.(*core.Predicate); ok {
				predicates = append(predicates, p)
			} else {
				// if this is an atomic or variable, something is wrong
				fmt.Printf("error parsing predicate: %v\n", err)
			}
			s = s[next:]
		}
		if len(predicates) > 0 {
			return &ParseResult{Predicates: predicates, Frame: frame, IsQuery: true}
		}
	} else if final == '.' {
		// typically a single ground fact
		frame := core.NewFrame()
		predicate, _, err := kb.ParseClause(s, nil, frame)
		if err != nil {
			fmt.Println("Error parsing fact: ", err)
			return nil
		}
		if predicate, ok := predicate.(*core.Predicate); ok {
			return &ParseResult{Predicates: []*core.Predicate{predicate}, Frame: frame, IsQuery: false}
		}
	} else if final == '!' {
		// an action
		// looks like a predicate but won't have a definition
	}
	return nil
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
				if i < len(s) {
					fmt.Println(" moving to ", i, s[i+1:])
				}
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
}
