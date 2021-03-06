package main

//go:generate msgp

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/tinylib/msgp/msgp"
)

// Search structure
type Search struct {
	mu sync.RWMutex

	Items map[string]map[string]string
}

// Result structure
//msgp:ignore Result
type Result struct {
	Key     string   `json:"key"`
	Found   []string `json:"found"`
	Count   int      `json:"count"`
	Start   int      `json:"start"`
	Stop    int      `json:"stop"`
	Elapsed string   `json:"elapsed"`
}

// NewSearch function
func NewSearch() *Search {
	return &Search{
		Items: make(map[string]map[string]string),
	}
}

// Sync function
func (s *Search) Sync(db *os.File, child bool) error {
	log.Debug("Search DB syncing...")

	if !child {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}

	t1 := time.Now()

	err := msgp.WriteFile(s, db)
	if err != nil {
		return err
	}

	log.Debug("Search DB write to disk", "duration", time.Since(t1).Round(time.Millisecond),
		"size", fmt.Sprintf("%d mb", s.Msgsize()/1024/1024))

	return db.Sync()
}

// Set function
func (s *Search) Set(key, id, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Items[key] == nil {
		s.Items[key] = make(map[string]string)
	}

	s.Items[key][id] = strings.ToLower(value)
}

// Del function
func (s *Search) Del(key, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Items[key] == nil {
		return
	}

	delete(s.Items[key], id)

	if len(s.Items[key]) == 0 {
		delete(s.Items, key)
	}
}

// Search function
func (s *Search) Search(key, query string, start, stop int) *Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &Result{
		Key:   key,
		Found: []string{},
		Count: 0,
	}

	if s.Items[key] == nil {
		return result
	}

	t1 := time.Now()

	query = strings.ToLower(query)
	for id, value := range s.Items[key] {
		if strings.Contains(value, query) {
			result.Found = append(result.Found, id)
		}
	}

	result.Elapsed = time.Since(t1).Round(time.Millisecond).String()

	if start > stop {
		start = stop
	}

	if start < 0 {
		start = 0
	}

	if stop > len(result.Found) || stop == 0 {
		stop = len(result.Found)
	}

	result.Start = start
	result.Stop = stop
	result.Count = len(result.Found)

	sort.Strings(result.Found)

	result.Found = result.Found[start:stop]

	return result
}

// Flush function
func (s *Search) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Items = make(map[string]map[string]string)
}
