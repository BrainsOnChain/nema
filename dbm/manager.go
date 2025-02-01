package dbm

import "github.com/brainsonchain/nema/nema"

type Manager struct {
}

func NewManager() *Manager {
	return &Manager{}
}

// InitiateSchema builds the schema for the database

func (m *Manager) GetNema() nema.Neuro {
	return nema.NewNeuro()
}
