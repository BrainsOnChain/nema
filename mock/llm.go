package mock

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

/*
	MockLLM implements the llms.Model interface.

    // GenerateContent asks the model to generate content from a sequence of
    // messages. It's the most general interface for multi-modal LLMs that support
    // chat-like interactions.
    GenerateContent(ctx context.Context, messages []MessageContent, options ...CallOption) (*ContentResponse, error)

    // Call is a simplified interface for a text-only Model, generating a single
    // string response from a single string prompt.
    //
    // Deprecated: this method is retained for backwards compatibility. Use the
    // more general [GenerateContent] instead. You can also use
    // the [GenerateFromSinglePrompt] function which provides a similar capability
    // to Call and is built on top of the new interface.
    Call(ctx context.Context, prompt string, options ...CallOption) (string, error)
*/

type MockLLM struct{}

func (m *MockLLM) GenerateContent(
	ctx context.Context,
	messages []llms.MessageContent,
	options ...llms.CallOption,
) (*llms.ContentResponse, error) {
	responseStr := `
		{
			"human_message": "Hello, world!",
			"motor_neurons": [
				{
					"neuron": "motor_neuron_1",
					"value": 1
				}
			],
			"sensory_neurons": [
				{
					"neuron": "sensory_neuron_1",
					"value": 1
				}
			],
			"changed": true
		}
	`

	return &llms.ContentResponse{Choices: []*llms.ContentChoice{
		{Content: responseStr},
	}}, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}
