package server

import (
	"encoding/json"
	"net/http"
)

// tweet returns
func (s *Server) tweet(w http.ResponseWriter, r *http.Request) {
	type resp struct {
		Message string `json:"message"`
	}

	jsonResp := resp{
		Message: "Tweeted",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jsonResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// tweetReply is a handler that replies to a tweet.
func (s *Server) tweetReply(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Message string `json:"message"`
	}

	var tweetReq req
	if err := json.NewDecoder(r.Body).Decode(&tweetReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	type resp struct {
		Message string `json:"message"`
	}

	jsonResp := resp{
		Message: "Replied",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jsonResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
