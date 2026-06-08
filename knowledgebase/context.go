package knowledgebase

import "github.com/jteutenberg/understate/core"

type SearchContext struct {
	subContext core.QueryContext
	depth      int
}

func NewSearchContext(subContext core.QueryContext) *SearchContext {
	ctx := &SearchContext{
		subContext: subContext,
		depth:      0,
	}
	return ctx
}

func (ctx *SearchContext) Done() <-chan struct{} {
	return ctx.subContext.Done()
}

func (ctx *SearchContext) Cancel() {
	ctx.subContext.Cancel()
}
