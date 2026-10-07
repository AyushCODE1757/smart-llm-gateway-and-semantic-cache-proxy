package router

import (
	"fmt"
	"os"
)

type Provider string

const (
	ProviderOllama Provider = "ollama"
	ProviderOpenAI Provider = "openai"
)

type Router struct {
	Provider  Provider
	OllamaURL string
	OpenAIURL string
	OpenAIKey string
}

func NewRouter() (*Router, error) {
	provider := Provider(os.Getenv("LLM_PROVIDER"))

	if provider == "" {
		provider = ProviderOllama
	}

	switch provider {
	case ProviderOllama, ProviderOpenAI:
	default:
		return nil, fmt.Errorf("unsupported LLM_PROVIDER: %s", provider)
	}

	return &Router{
		Provider:  provider,
		OllamaURL: os.Getenv("OLLAMA_URL"),
		OpenAIURL: os.Getenv("OPENAI_URL"),
		OpenAIKey: os.Getenv("OPENAI_API_KEY"),
	}, nil
}

func (r *Router) URL() string {
	switch r.Provider {
	case ProviderOpenAI:
		return r.OpenAIURL
	default:
		return r.OllamaURL
	}
}
