package fakeserver

import (
	"encoding/json"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var globalCounter atomic.Int64

func init() {
	globalCounter.Store(10000)
}

// NextID returns the next unique numeric ID as a string.
func NextID() string {
	return strconv.FormatInt(globalCounter.Add(1), 10)
}

// Store is a generic, thread-safe, in-memory document store keyed by string ID.
// Each document is stored as json.RawMessage (raw JSON bytes).
type Store struct {
	mu    sync.RWMutex
	items map[string]json.RawMessage
	order []string // insertion order for deterministic listing
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{
		items: make(map[string]json.RawMessage),
	}
}

// Create adds a new document with an auto-generated ID, sets standard fields
// (id, self, creationTime, lastUpdated), and returns the ID and stored doc.
func (s *Store) Create(body json.RawMessage, selfURL string) (string, json.RawMessage) {
	id := NextID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	doc := mergeFields(body, map[string]any{
		"id":           id,
		"self":         selfURL + "/" + id,
		"creationTime": now,
		"lastUpdated":  now,
	})
	s.mu.Lock()
	s.items[id] = doc
	s.order = append(s.order, id)
	s.mu.Unlock()
	return id, doc
}

// CreateWithID stores a document with a caller-provided ID (for special cases like external IDs).
func (s *Store) CreateWithID(id string, doc json.RawMessage) {
	s.mu.Lock()
	if _, exists := s.items[id]; !exists {
		s.order = append(s.order, id)
	}
	s.items[id] = doc
	s.mu.Unlock()
}

// Get returns a document by ID. Returns nil, false if not found.
func (s *Store) Get(id string) (json.RawMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.items[id]
	return doc, ok
}

// Update merges the patch into the existing document and updates "lastUpdated".
// Returns the updated doc and true, or nil and false if not found.
func (s *Store) Update(id string, patch json.RawMessage) (json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.items[id]
	if !ok {
		return nil, false
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	merged := mergeJSON(existing, patch)
	merged = mergeFields(merged, map[string]any{
		"lastUpdated": now,
	})
	s.items[id] = merged
	return merged, true
}

// Delete removes a document by ID. Returns true if it existed.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[id]
	if !ok {
		return false
	}
	delete(s.items, id)
	for i, oid := range s.order {
		if oid == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// List returns all documents in insertion order.
func (s *Store) List() []json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]json.RawMessage, 0, len(s.order))
	for _, id := range s.order {
		if doc, ok := s.items[id]; ok {
			result = append(result, doc)
		}
	}
	return result
}

// Count returns the number of stored documents.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// mergeJSON merges two JSON objects. Fields in patch override those in base.
func mergeJSON(base, patch json.RawMessage) json.RawMessage {
	var baseMap, patchMap map[string]json.RawMessage
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return patch
	}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return base
	}
	for k, v := range patchMap {
		baseMap[k] = v
	}
	out, _ := json.Marshal(baseMap)
	return out
}

// mergeFields merges a map of fields into a JSON document.
func mergeFields(doc json.RawMessage, fields map[string]any) json.RawMessage {
	patch, _ := json.Marshal(fields)
	return mergeJSON(doc, patch)
}
