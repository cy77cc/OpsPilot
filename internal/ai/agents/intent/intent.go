package intent

import "context"

type Router struct {
	rules RuleLayer
	model ModelLayer
}

func NewRouter() *Router {
	return &Router{
		rules: RuleLayer{},
		model: ModelLayer{},
	}
}

func (r *Router) Route(ctx context.Context, message string) (Decision, error) {
	if decision, ok := r.rules.Classify(message); ok {
		return decision, nil
	}
	return r.model.Classify(ctx, message)
}
