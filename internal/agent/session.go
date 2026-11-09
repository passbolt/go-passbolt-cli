package agent

import (
	"context"
	"sort"
	"time"
)

const IdleTimeout = time.Hour

type session struct {
	deadline time.Time
	close    func(context.Context)
}

// Sessions is owned by the serialized dispatcher. Status never extends a lease.
type Sessions struct {
	now     func() time.Time
	entries map[string]session
}

func NewSessions(now func() time.Time) *Sessions {
	return &Sessions{now: now, entries: make(map[string]session)}
}
func (s *Sessions) Put(key string, close func(context.Context)) {
	s.entries[key] = session{s.now().Add(IdleTimeout), close}
}
func (s *Sessions) Touch(key string) bool {
	entry, ok := s.entries[key]
	if !ok || !s.now().Before(entry.deadline) {
		return false
	}
	entry.deadline = s.now().Add(IdleTimeout)
	s.entries[key] = entry
	return true
}
func (s *Sessions) Expire(ctx context.Context) {
	for key, entry := range s.entries {
		if !s.now().Before(entry.deadline) {
			s.Lock(ctx, key)
		}
	}
}
func (s *Sessions) Lock(ctx context.Context, key string) {
	if entry, ok := s.entries[key]; ok {
		delete(s.entries, key)
		entry.close(ctx)
	}
}
func (s *Sessions) Close(ctx context.Context) {
	for key := range s.entries {
		s.Lock(ctx, key)
	}
}

type Status struct {
	Account string    `json:"account"`
	Expires time.Time `json:"expires"`
}

func (s *Sessions) Status() []Status {
	result := []Status{}
	for key, entry := range s.entries {
		result = append(result, Status{key, entry.deadline})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Account < result[j].Account })
	return result
}
