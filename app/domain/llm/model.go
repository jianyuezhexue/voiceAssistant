package llm

type LLM struct {
}

func NewLLM() LLMInterface {
	return &LLM{}
}
