package knowledgebase

import (
	"strings"

	"github.com/jteutenberg/understate/core"
)

type SearchContext struct {
	subContext core.QueryContext
	history    []string
	depth      int
}

func NewSearchContext(subContext core.QueryContext) *SearchContext {
	ctx := &SearchContext{
		subContext: subContext,
		depth:      0,
		history:    make([]string, 0, 100),
	}
	return ctx
}

func (ctx *SearchContext) Done() <-chan struct{} {
	return ctx.subContext.Done()
}

func (ctx *SearchContext) Cancel() {
	ctx.subContext.Cancel()
}

func (ctx *SearchContext) historyString(p *core.Predicate) string {
	sb := strings.Builder{}
	sb.WriteString(p.Definition.Functor)
	sb.WriteString("(")
	sb.WriteString(p.CanonicalArgsString(0))
	sb.WriteString(")")
	return sb.String()
}

func (ctx *SearchContext) AddHistory(p *core.Predicate) {
	s := ctx.historyString(p)
	ctx.history = append(ctx.history, s)
}

func (ctx *SearchContext) PopHistory() {
	ctx.history = ctx.history[:len(ctx.history)-1]
}

func (ctx *SearchContext) InHistory(p *core.Predicate) bool {
	s := ctx.historyString(p)
	for _, h := range ctx.history {
		if h == s {
			return true
		}
	}
	return false
}
