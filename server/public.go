package server

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// nemaState is a handler that returns the current state of the nema.
func (s *Server) nemaState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.nemaManager.GetState()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// TODO: Implement this
// nemaPrompt is a handler that takes a incoming prompt, asks the LLM, and
// returns the response.
func (s *Server) nemaPrompt(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	var prompt struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.log.Info("incoming prompt", zap.String("prompt", prompt.Prompt))

	response, err := s.nemaManager.AskLLM(r.Context(), prompt.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type resp struct {
		HumanMessage string `json:"human_message"`
	}

	jsonResp := resp{
		HumanMessage: response.HumanMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jsonResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
