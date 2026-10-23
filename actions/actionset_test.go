package actions_test

import (
	"bufio"
	"strings"
	"testing"

	"github.com/jteutenberg/understate/actions"
	"github.com/jteutenberg/understate/core"
	"github.com/jteutenberg/understate/io"
	"github.com/jteutenberg/understate/knowledgebase"
)

// moveDomain is the blocks-world move action:
//
//	:at(X:Thing, Y:Location).
//	:move(X:Thing, From:Location, To:Location).
//	preconditions: at(X, From)
//	effects: not(at(X, From)), at(X, To)
const moveDomain = `
:at(X:Thing, Y:Location).
:move(X:Thing, From:Location, To:Location).

move(X, From, To)
| at(X, From)
| not(at(X, From)), at(X, To).

at(ball, kitchen).
at(box, hall).
`

func loadDomain(t *testing.T, script string) (*knowledgebase.KnowledgeBase, *actions.ActionSet) {
	t.Helper()
	kb := knowledgebase.NewKnowledgeBase()
	set := actions.NewActionSet()
	parser := io.NewPredicateReader(
		[]byte{knowledgebase.ActionSeparator, knowledgebase.RuleSeparator},
		[]byte{knowledgebase.AssertTerminator, knowledgebase.CommandTerminator, knowledgebase.QueryTerminator},
	)
	tokens := parser.Parse(bufio.NewReader(strings.NewReader(script)))
	for token := range tokens {
		_, _, actionDefinition, _, err := kb.Process(token)
		if err != nil {
			t.Fatalf("parse %q: %v", token.Predicates, err)
		}
		if actionDefinition != nil {
			set.AddAction(actionDefinition)
		}
	}
	return kb, set
}

func mustPredicate(t *testing.T, kb *knowledgebase.KnowledgeBase, text string) *core.Predicate {
	t.Helper()
	parser := io.NewPredicateReader(
		[]byte{knowledgebase.ActionSeparator, knowledgebase.RuleSeparator},
		[]byte{knowledgebase.AssertTerminator, knowledgebase.CommandTerminator, knowledgebase.QueryTerminator},
	)
	tokens := parser.Parse(bufio.NewReader(strings.NewReader(text)))
	token, ok := <-tokens
	if !ok {
		t.Fatalf("no predicate in %q", text)
	}
	query, command, actionDefinition, _, err := kb.Process(token)
	for range tokens {
	}
	if err != nil {
		t.Fatalf("parse %q: %v", text, err)
	}
	if actionDefinition != nil {
		t.Fatalf("parse %q: got an action definition", text)
	}
	if command != nil {
		return command
	}
	if len(query) != 1 {
		t.Fatalf("parse %q: expected 1 predicate, got %d", text, len(query))
	}
	return query[0]
}

func holds(t *testing.T, kb *knowledgebase.KnowledgeBase, text string) bool {
	t.Helper()
	return kb.Exists(mustPredicate(t, kb, text), core.NewQueryContext(), core.NewSearchHistory())
}

func TestMoveActionPreconditionsAndEffects(t *testing.T) {
	kb, set := loadDomain(t, moveDomain)
	act := set.GetAction(mustPredicate(t, kb, "move(X, From, To)?"))
	if act == nil {
		t.Fatal("move action was not added")
	}
	if act.Signature.String() != "move(X, From, To)" {
		t.Errorf("signature: %s", act.Signature)
	}
	if len(act.PositivePreconditions) != 1 || act.PositivePreconditions[0].String() != "at(X, From)" {
		t.Errorf("preconditions: %v", act.PositivePreconditions)
	}
	if len(act.NegativePreconditions) != 0 {
		t.Errorf("unexpected negative preconditions: %v", act.NegativePreconditions)
	}
	if len(act.DeleteEffects) != 1 || act.DeleteEffects[0].String() != "at(X, From)" {
		t.Errorf("delete effects: %v", act.DeleteEffects)
	}
	if len(act.AddEffects) != 1 || act.AddEffects[0].String() != "at(X, To)" {
		t.Errorf("add effects: %v", act.AddEffects)
	}
}

func TestGetActionUnifiesMove(t *testing.T) {
	kb, set := loadDomain(t, moveDomain)
	if set.GetAction(mustPredicate(t, kb, "at(ball, kitchen)?")) != nil {
		t.Error("at/2 matched the move action")
	}

	toHall := set.GetAction(mustPredicate(t, kb, "move(ball, kitchen, hall)?"))
	toGarden := set.GetAction(mustPredicate(t, kb, "move(ball, kitchen, garden)?"))
	if toHall == nil || toGarden == nil {
		t.Fatal("expected both ground moves")
	}
	if toHall.Signature.String() != "move(ball, kitchen, hall)" {
		t.Errorf("signature: %s", toHall.Signature)
	}
	if toGarden.Signature.String() != "move(ball, kitchen, garden)" {
		t.Errorf("second grounding changed the first, or failed to unify: %s", toGarden.Signature)
	}
	if toHall.PositivePreconditions[0].String() != "at(ball, kitchen)" {
		t.Errorf("precondition: %s", toHall.PositivePreconditions[0])
	}
	if toHall.DeleteEffects[0].String() != "at(ball, kitchen)" {
		t.Errorf("delete effect: %s", toHall.DeleteEffects[0])
	}
	if toHall.AddEffects[0].String() != "at(ball, hall)" {
		t.Errorf("add effect: %s", toHall.AddEffects[0])
	}
	if !toHall.DeleteEffects[0].IsFact() || !toHall.AddEffects[0].IsFact() {
		t.Error("ground move effects should be facts")
	}
}

func TestMoveIndexedByAtEffects(t *testing.T) {
	kb, set := loadDomain(t, moveDomain)
	holding := &core.Predicate{Definition: &core.PredicateDefinition{Functor: "holding"}}
	if got := set.GetActionsByPositiveEffect(holding); len(got) != 0 {
		t.Errorf("holding matched %d actions", len(got))
	}
	if got := set.GetActionsByNegativeEffect(holding); len(got) != 0 {
		t.Errorf("holding matched %d delete actions", len(got))
	}

	added := set.GetActionsByPositiveEffect(mustPredicate(t, kb, "at(ball, hall)?"))
	if len(added) != 1 {
		t.Fatalf("positive at/2 matches: %d", len(added))
	}
	if added[0].Signature.String() != "move(ball, From, hall)" {
		t.Errorf("positive effect grounding: %s", added[0].Signature)
	}
	if added[0].AddEffects[0].String() != "at(ball, hall)" {
		t.Errorf("add effect: %s", added[0].AddEffects[0])
	}
	if added[0].DeleteEffects[0].String() != "at(ball, From)" {
		t.Errorf("delete effect: %s", added[0].DeleteEffects[0])
	}

	deleted := set.GetActionsByNegativeEffect(mustPredicate(t, kb, "at(box, hall)?"))
	if len(deleted) != 1 {
		t.Fatalf("negative at/2 matches: %d", len(deleted))
	}
	if deleted[0].Signature.String() != "move(box, hall, To)" {
		t.Errorf("negative effect grounding: %s", deleted[0].Signature)
	}
	if deleted[0].DeleteEffects[0].String() != "at(box, hall)" {
		t.Errorf("delete effect: %s", deleted[0].DeleteEffects[0])
	}
	if deleted[0].AddEffects[0].String() != "at(box, To)" {
		t.Errorf("add effect: %s", deleted[0].AddEffects[0])
	}
}

func TestApplicableMoves(t *testing.T) {
	kb, set := loadDomain(t, moveDomain)

	got := map[string]*actions.Action{}
	for act := range set.ApplicableActions(kb) {
		got[act.Signature.String()] = act
	}
	ball := got["move(ball, kitchen, To)"]
	box := got["move(box, hall, To)"]
	if ball == nil || box == nil || len(got) != 2 {
		t.Fatalf("applicable moves: %v", keys(got))
	}
	if ball.PositivePreconditions[0].String() != "at(ball, kitchen)" {
		t.Errorf("ball precondition: %s", ball.PositivePreconditions[0])
	}
	if ball.DeleteEffects[0].String() != "at(ball, kitchen)" || !ball.DeleteEffects[0].IsFact() {
		t.Errorf("ball delete effect: %s", ball.DeleteEffects[0])
	}
	if ball.AddEffects[0].String() != "at(ball, To)" || ball.AddEffects[0].IsFact() {
		t.Errorf("ball add effect should stay open: %s", ball.AddEffects[0])
	}

	if !set.GetAction(mustPredicate(t, kb, "move(ball, kitchen, hall)?")).IsApplicable(kb) {
		t.Error("ball can move from the kitchen")
	}
	if set.GetAction(mustPredicate(t, kb, "move(ball, hall, kitchen)?")).IsApplicable(kb) {
		t.Error("ball is not in the hall")
	}
}

func TestApplyMoveUpdatesAt(t *testing.T) {
	kb, set := loadDomain(t, moveDomain)
	move := set.GetAction(mustPredicate(t, kb, "move(ball, kitchen, hall)?"))
	if move == nil || !move.IsApplicable(kb) {
		t.Fatal("move(ball, kitchen, hall) should be applicable")
	}
	move.ApplyTo(kb.State)

	if !holds(t, kb, "at(ball, hall)?") {
		t.Error("ball should be in the hall")
	}
	if holds(t, kb, "at(ball, kitchen)?") {
		t.Error("ball should have left the kitchen")
	}
	if holds(t, kb, "at(box, kitchen)?") {
		t.Error("box should have stayed in the hall")
	}
	if !holds(t, kb, "at(box, hall)?") {
		t.Error("box should still be in the hall")
	}
	if set.GetAction(mustPredicate(t, kb, "move(ball, kitchen, hall)?")).IsApplicable(kb) {
		t.Error("ball can no longer move from the kitchen")
	}
	if !set.GetAction(mustPredicate(t, kb, "move(ball, hall, kitchen)?")).IsApplicable(kb) {
		t.Error("ball can move back from the hall")
	}
}

func keys(actions map[string]*actions.Action) []string {
	out := make([]string, 0, len(actions))
	for name := range actions {
		out = append(out, name)
	}
	return out
}
