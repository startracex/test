package concurrent

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"iter"
	"slices"
	"sync"
)

type Slice[T any] struct {
	mu sync.RWMutex
	s  []T
}

func NewSlice[T any](n int) *Slice[T] {
	return &Slice[T]{
		s: make([]T, 0, n),
	}
}

func (s *Slice[T]) Get(idx int) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if idx < 0 || idx >= len(s.s) {
		var zero T
		return zero, false
	}
	return s.s[idx], true
}

func (s *Slice[T]) Append(val T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.s = append(s.s, val)
}

func (s *Slice[T]) Delete(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.s) {
		return
	}
	s.s = slices.Delete(s.s, idx, idx+1)
}

func (s *Slice[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.s)
}

func (s *Slice[T]) Cap() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cap(s.s)
}

func (s *Slice[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.s = s.s[:0]
}

func (s *Slice[T]) Clone() *Slice[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &Slice[T]{
		s: slices.Clone(s.s),
	}
}

func (s *Slice[T]) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprint(s.s)
}

func (s *Slice[T]) Slice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.s)
}

func (s *Slice[T]) Iterator() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, v := range s.s {
			if !yield(v) {
				return
			}
		}
	}
}

func (s *Slice[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	input := s.s
	if input == nil {
		input = make([]T, 0)
	}
	return json.MarshalEncode(enc, input)
}

func (s *Slice[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.s = make([]T, 0)
	return json.UnmarshalDecode(dec, &s.s)
}
