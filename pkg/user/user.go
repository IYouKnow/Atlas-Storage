package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// User represents a system user.
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	// Add permissions later if needed
}

// Store manages user persistence.
type Store struct {
	mu       sync.RWMutex
	filePath string
	Users    map[string]*User `json:"users"`
}

// NewStore creates a new user store backed by the given file path.
// It loads existing users if the file exists.
func NewStore(path string) (*Store, error) {
	s := &Store{
		filePath: path,
		Users:    make(map[string]*User),
	}

	if err := s.load(); err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	// We define a temporary struct to match the JSON structure expected:
	// either root level map, or a list.
	// The plan said "users.json", usually a list or map.
	// Let's use a simple map for direct lookup by username during auth.
	// But for JSON editability, a list is sometimes nicer?
	// User said "facilidade de poder editar o ficheiro manualmente".
	// A map `{"username": {"password_hash": "..."}}` is easy to read.
	// Or a list `[{"username": "...", ...}]`.
	// Let's stick effectively to a map serialized, or a list serialized to a map in memory.
	// Let's support a simple map structure in JSON for O(1) lookups and easy editing.
	// { "users": { "bob": { ... } } } - maybe too nested.
	// Simple map: { "bob": { "username": "bob", "password_hash": "..." } }

	return json.Unmarshal(data, &s.Users)
}

// Save persists the users to disk.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.Users, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// Add creates a new user.
func (s *Store) Add(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.Users[username]; exists {
		return fmt.Errorf("user %s already exists", username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.Users[username] = &User{
		Username:     username,
		PasswordHash: string(hash),
	}

	return nil // Caller must call Save() explicitly to persist? Or we do it here?
	// Better to separate concern, but for a CLI command "Add", we expect instant persistence.
	// Let's not call Save() inside Add() to allow bulk ops, but for CLI usage we will call Add then Save.
}

// Delete removes a user.
func (s *Store) Delete(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Users, username)
}

// Authenticate verifies password for a user.
func (s *Store) Authenticate(username, password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.Users[username]
	if !ok {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// List returns all usernames.
func (s *Store) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.Users))
	for k := range s.Users {
		keys = append(keys, k)
	}
	return keys
}
