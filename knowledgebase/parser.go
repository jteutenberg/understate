package knowledgebase

import (
	"fmt"
	"io"

	"github.com/jteutenberg/understate/actions"
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/rules"
	"github.com/jteutenberg/understate/state"
)

type ParseResult struct {
	Predicates []*core.Predicate
	Rule       *rules.Rule
	Action     *actions.Action
	isQuery    bool
}

func (kb *KnowledgeBase) Parse(reader io.ByteReader) <-chan ParseResult {
	result := make(chan ParseResult)
	go func() {
		line := make([]byte, 0, 1000)
		for {
			// consume until a . or ?
			isQuery := false
			isRule := false
			prev := byte(' ')
			for {
				if b, err := reader.ReadByte(); err != nil {
					return
				} else {
					if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
						continue
					}
					if b == '.' {
						break
					}
					if b == '?' {
						isQuery = true
						break
					}
					if prev == ':' && b == '-' {
						isRule = true
					}

					line = append(line, b)
					prev = b
				}
			}
			s := string(line)
			// did we see a rule's ":-"?
			if isRule {
				if isQuery {
					// Error. Rule definitions should not be queries
				}
				if r, err := kb.ParseRule(s); err == nil {
					result <- ParseResult{Rule: r, isQuery: false}
				}
			}
			// somehow check if this is an action definition
			//result <- ParseResult{Predicates: make([]*core.Predicate, 0, 5), isQuery: isQuery}

			// anything else is a set of Predicates with a shared Frame
		}
	}()
	return result
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
