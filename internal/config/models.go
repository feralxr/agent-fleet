package config

// DefaultModels returns a curated model list for a known agent.
// Users can override / extend via fleet.toml [[agents]] models = [...]
// or type any id in the picker.
func DefaultModels(id AgentID) []string {
	switch id {
	case AgentClaude:
		return []string{
			"default",
			"sonnet",
			"opus",
			"haiku",
			"fable",
			"best",
			"claude-sonnet-4-5",
			"claude-opus-4-5",
			"claude-haiku-4-5",
		}
	case AgentCursor:
		return []string{
			"auto",
			"composer-1",
			"gpt-5",
			"gpt-5-mini",
			"claude-sonnet-4-5",
			"claude-opus-4-5",
			"gemini-2.5-pro",
		}
	case AgentOpenCode:
		return []string{
			"anthropic/claude-sonnet-4-5",
			"anthropic/claude-opus-4-5",
			"openai/gpt-5",
			"openai/gpt-5-mini",
			"google/gemini-2.5-pro",
			"opencode/gpt-5",
		}
	case AgentGrok:
		return []string{
			"grok-4",
			"grok-4.6",
			"grok-3",
			"grok-3-mini",
		}
	default:
		return []string{}
	}
}

// ModelFlag is the CLI flag used to pass a model when launching the agent.
func ModelFlag(id AgentID) string {
	switch id {
	case AgentOpenCode, AgentGrok:
		return "--model" // also accept -m; long form is universal
	default:
		return "--model"
	}
}
