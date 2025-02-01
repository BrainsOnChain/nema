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
	nemaState, err := dbm.GetState()
	if err != nil {
		if errors.Is(err, errNoState) {
			log.Info("No state found, creating new Nema")
			nemaState = NewNeuro()
		} else {
			return nil, fmt.Errorf("error getting Nema: %w", err)
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

func (m *Manager) AskLLM(ctx context.Context, prompt string) (string, error) {
	m.messages = append(m.messages, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

	completion, err := m.llm.GenerateContent(ctx, m.messages, llms.WithTemperature(1))
	if err != nil {
		return "", fmt.Errorf("error generating completion: %w", err)
	}

	response := completion.Choices[0].Content

	// Strip markdown code fence if present
	response = strings.TrimPrefix(response, "```json\n")
	response = strings.TrimSuffix(response, "\n```")

	fmt.Println(response)

	m.messages = append(m.messages, llms.TextParts(llms.ChatMessageTypeAI, response))

	// Unmarshal the response into a neuro struct
	type resp struct {
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

	var r resp
	if err = json.Unmarshal([]byte(response), &r); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Update the neurons if the response indicates that they have changed
	if r.Changed {
		m.log.Info("Neurons changed, updating state")
		for _, neuron := range r.MotorNeurons {
			m.state.MotorNeurons[neuron.Neuron] = neuron.Value
		}
		for _, neuron := range r.SensoryNeurons {
			m.state.SensoryNeurons[neuron.Neuron] = neuron.Value
		}

		// Update the state
		id, err := m.db.SaveState(m.state)
		if err != nil {
			return "", fmt.Errorf("error updating state: %w", err)
		}

		// Save the prompt and response
		if err := m.db.SavePrompt(id, m.initialPrompt, response); err != nil {
			return "", fmt.Errorf("error saving prompt: %w", err)
		}
	} else {
		m.log.Info("No neurons changed, skipping update")
	}

	return response, nil
}
