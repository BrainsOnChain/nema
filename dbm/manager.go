package dbm

import "github.com/brainsonchain/nema/nema"

type Manager struct {
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) GetNema() nema.Neuro {
	return nema.NewNeuro()
}
