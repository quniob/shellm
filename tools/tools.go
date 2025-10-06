package tools

import (
	"context"
	"encoding/json"
	"github.com/openai/openai-go/v2"
)

type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Call(ctx context.Context, raw json.RawMessage) (string, error)
}

type Registry struct{ m map[string]Tool }

func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{m: map[string]Tool{}}
	for _, t := range tools {
		r.m[t.Name()] = t
	}
	return r
}
func (r *Registry) Get(name string) (Tool, bool) { t, ok := r.m[name]; return t, ok }

func (r *Registry) Tools() []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(r.m))
	for _, t := range r.m {
		out = append(out, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: openai.FunctionDefinitionParam{
					Name:        t.Name(),
					Description: openai.String(t.Description()),
					Parameters:  t.Schema(),
				},
			},
		})
	}
	return out
}
