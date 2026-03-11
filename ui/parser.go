package ui

import (
	"fmt"

	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/state"
)

type Parser struct {
	predicateDefinitions map[string]*core.PredicateDefinition
	state                *state.State
}

func NewParser() *Parser {
	return &Parser{
		predicateDefinitions: make(map[string]*core.PredicateDefinition),
		state:                state.NewState(),
	}
}

func (p *Parser) AddPredicateDefinition(pdef *core.PredicateDefinition) {
	p.predicateDefinitions[pdef.Functor] = pdef
}

func (p *Parser) ParseArguments(s string) ([]core.Unifiable, error) {
	args := make([]core.Unifiable, 0, 5)
	for i := 0; i < len(s); {
		c, next, err := p.ParseClause(s[i:], nil)
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

func (p *Parser) ParseClause(s string, typeHint *core.Type) (core.Unifiable, int, error) {
	// consume the lead string up to the first '('',', ')', or end of string
	for i := 0; i < len(s); i++ {
		if s[i] == '(' {
			// this will be a predicate. Parse its arguments
			functor := s[:i]
			pdef, ok := p.predicateDefinitions[functor]
			if !ok {
				//TODO: create a new predicate definition
				return nil, 0, fmt.Errorf("unknown predicate: %s", functor)
			}
			// find the close parenthesis
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
			//j is one after the close parenthesis, args end two char earlier
			args, err := p.ParseArguments(s[i+1 : j-1])
			if err != nil {
				return nil, j, err
			}
			// skip any following comma
			for j < len(s) && (s[j] == ' ' || s[j] == ',') {
				j++
			}
			if len(args) != len(pdef.ArgDefinitions) {
				return nil, 0, fmt.Errorf("expected %d arguments, got %d", len(pdef.ArgDefinitions), len(args))
			}
			vrefs := make([]*core.VariableReference, len(args))
			for i, argDef := range pdef.ArgDefinitions {
				vrefs[i] = &core.VariableReference{
					Label: argDef.Label,
					Ref:   nil,
				}
			}
			return &core.Predicate{
				Definition: pdef,
				VarLabels:  make([]string, 0, 5),
				VarRefs:    vrefs,
			}, j, nil
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
				return p.state.GetAtomic(s[:i], typeHint), j, nil
			} else {
				// variable
				return &core.VariableReference{
					Label: s[:i],
					Ref:   nil,
				}, i + 1, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("invalid clause: %s", s)
}
