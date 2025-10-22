package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State stores the last known recommendation and run timestamp.
type State struct {
	LastRecommendation *int       `json:"last_recommendation,omitempty"`
	LastRun            *time.Time `json:"last_run,omitempty"`
}

// Store manages persistence of application state.
type Store struct {
	path string
}

// NewStore ensures the state directory exists and returns a Store that reads
// and writes a JSON file named state.json within that directory.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure state dir: %w", err)
	}
	return &Store{path: filepath.Join(dir, "state.json")}, nil
}

// Load reads persisted state. If no state exists yet, it returns an empty State.
func (s *Store) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state: %w", err)
	}

	return state, nil
}

// Save persists the supplied state atomically.
func (s *Store) Save(state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace state: %w", err)
	}

	return nil
}
