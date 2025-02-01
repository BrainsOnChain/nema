package nema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"go.uber.org/zap"
)

type Manager struct {
	log           *zap.Logger
	db            *dbm
	state         neuro
	initialPrompt string
	llm           llms.Model
	messages      []llms.MessageContent
}

func NewManager(log *zap.Logger, dbm *dbm, initialPrompt string, llm llms.Model) (*Manager, error) {

	// Get the initial state
	nemaState, err := dbm.getState()
	if err != nil {
		if errors.Is(err, errNoState) {
			log.Info("no state found, creating new nema")
			nemaState = NewNeuro()
		} else {
			return nil, fmt.Errorf("error getting nema: %w", err)
		}
	}

	// Build the initial prompt
	initialPrompt = strings.Replace(initialPrompt, "%s", nemaState.JSONString(), 1)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, initialPrompt),
	}

	return &Manager{
		log:           log,
		db:            dbm,
		state:         nemaState,
		initialPrompt: initialPrompt,
		llm:           llm,
		messages:      messages,
	}, nil
}

func (m *Manager) GetState() neuro {
	return m.state
}

func (m *Manager) AskLLM(ctx context.Context, prompt string) (llmResponse, error) {
	m.messages = append(m.messages, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

	completion, err := m.llm.GenerateContent(ctx, m.messages, llms.WithTemperature(1))
	if err != nil {
		return llmResponse{}, fmt.Errorf("error generating completion: %w", err)
	}

	response := completion.Choices[0].Content

	// Strip the response to only the JSON object. Everything before ```json and
	// after ``` is removed.
	response = strings.TrimPrefix(response, "```json\n")
	response = strings.TrimSuffix(response, "\n```")

	m.messages = append(m.messages, llms.TextParts(llms.ChatMessageTypeAI, response))

	var lr llmResponse
	if err = json.Unmarshal([]byte(response), &lr); err != nil {
		return llmResponse{}, fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Update the neurons if the response indicates that they have changed
	if lr.Changed {
		m.log.Info("neurons changed, updating state")
		for _, neuron := range lr.MotorNeurons {
			m.state.updateMotorNeuron(neuron.Neuron, neuron.Value)
		}
		for _, neuron := range lr.SensoryNeurons {
			m.state.updateSensoryNeuron(neuron.Neuron, neuron.Value)
		}

		// Update the state
		id, err := m.db.saveState(m.state)
		if err != nil {
			return llmResponse{}, fmt.Errorf("error updating state: %w", err)
		}

		// Save the prompt and response
		if err := m.db.savePrompt(id, prompt, lr); err != nil {
			return llmResponse{}, fmt.Errorf("error saving prompt: %w", err)
		}
	} else {
		m.log.Info("no neurons changed, skipping update")
	}

	m.log.Info("response", zap.Any("response", lr))

	return lr, nil
}

type llmResponse struct {
	HumanMessage string `json:"human_message"`
	MotorNeurons []struct {
		Neuron string `json:"neuron"`
		Value  int    `json:"value"`
	} `json:"motor_neurons"`
	SensoryNeurons []struct {
		Neuron string `json:"neuron"`
		Value  int    `json:"value"`
	} `json:"sensory_neurons"`
	Changed bool `json:"changed"`
}
