package nema

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type dbm struct {
	db *sql.DB
}

func NewDBManager(dataSourceName string) (*dbm, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	return &dbm{db: db}, nil
}

// Initiate builds the schema for the database
func (m *dbm) Initiate() error {
	schema := /* sql */ `
	CREATE TABLE IF NOT EXISTS neural_states (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		state_count     INTEGER   NOT NULL,
		updated_at      TIMESTAMP NOT NULL,
		motor_neurons   TEXT      NOT NULL,     -- JSON string of motor neuron states
		sensory_neurons TEXT      NOT NULL    -- JSON string of sensory neuron states
	);

	CREATE INDEX IF NOT EXISTS idx_neural_states_updated_at ON neural_states(updated_at);

	CREATE TABLE IF NOT EXISTS prompts (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		neural_state_id INTEGER NOT NULL,
		question        TEXT NOT NULL,
		response        TEXT NOT NULL,
		completed_at    TIMESTAMP NOT NULL,

		FOREIGN KEY(neural_state_id) REFERENCES neural_states(id)
	);

	CREATE INDEX IF NOT EXISTS idx_prompts_neural_state_id ON prompts(neural_state_id);
	`

	// Execute the schema creation
	if _, err := m.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// SaveState saves the neural state to the database. It returns the ID of the
// state.
func (m *dbm) SaveState(n neuro) (int, error) {
	// Marshal the neuron maps to JSON strings
	motorJSON, err := json.Marshal(n.MotorNeurons)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal motor neurons: %w", err)
	}
	sensoryJSON, err := json.Marshal(n.SensoryNeurons)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal sensory neurons: %w", err)
	}

	q := /* sql */ `
		INSERT INTO neural_states
			(state_count, updated_at, motor_neurons, sensory_neurons)
		VALUES (?, ?, ?, ?)
		RETURNING id
	`

	var id int
	if err := m.db.QueryRow(q, n.StateCount, n.UpdatedAt, string(motorJSON), string(sensoryJSON)).Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to save nema: %w", err)
	}

	return id, nil
}

// SavePrompt saves the prompt to the database
func (m *dbm) SavePrompt(stateID int, prompt string, response string) error {
	q := /* sql */ `
		INSERT INTO prompts
			(neural_state_id, question, response, completed_at)
		VALUES (?, ?, ?, ?)
	`

	if _, err := m.db.Exec(q, stateID, prompt, response, time.Now()); err != nil {
		return fmt.Errorf("failed to save prompt: %w", err)
	}

	return nil
}

var errNoState = errors.New("no state found")

// GetState gets the neural state from the database
func (m *dbm) GetState() (neuro, error) {
	q := /* sql */ `
		SELECT state_count, updated_at, motor_neurons, sensory_neurons
		FROM neural_states
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var n neuro
	var motorJSON, sensoryJSON string

	err := m.db.QueryRow(q).Scan(&n.StateCount, &n.UpdatedAt, &motorJSON, &sensoryJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return neuro{}, errNoState
		}
		return neuro{}, fmt.Errorf("failed to get state: %w", err)
	}

	// Unmarshal the JSON strings back into maps
	if err := json.Unmarshal([]byte(motorJSON), &n.MotorNeurons); err != nil {
		return neuro{}, fmt.Errorf("failed to unmarshal motor neurons: %w", err)
	}
	if err := json.Unmarshal([]byte(sensoryJSON), &n.SensoryNeurons); err != nil {
		return neuro{}, fmt.Errorf("failed to unmarshal sensory neurons: %w", err)
	}

	return n, nil
}
