package concurrent

import (
	"encoding/json/v2"
	"slices"
	"sync"
	"testing"
)

func TestSliceZeroValueIsReadyToUse(t *testing.T) {
	var s Slice[int]

	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
	if _, ok := s.Get(0); ok {
		t.Error("Get(0) on an empty slice should report false")
	}

	s.Append(1)
	if s.Len() != 1 {
		t.Errorf("Len() after Append = %d, want 1", s.Len())
	}
	if v, ok := s.Get(0); !ok || v != 1 {
		t.Errorf("Get(0) = %d,%v want 1,true", v, ok)
	}
}

func TestSliceAppendAndGet(t *testing.T) {
	s := NewSlice[int](2)
	for i := 0; i < 5; i++ {
		s.Append(i)
	}

	if s.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", s.Len())
	}
	if s.Cap() < 5 {
		t.Errorf("Cap() = %d, want >= 5", s.Cap())
	}

	for i := 0; i < 5; i++ {
		if v, ok := s.Get(i); !ok || v != i {
			t.Errorf("Get(%d) = %d,%v want %d,true", i, v, ok, i)
		}
	}
}

func TestSliceGetOutOfRange(t *testing.T) {
	s := NewSlice[int](0)
	s.Append(1)

	for _, idx := range []int{-1, 1, 100} {
		v, ok := s.Get(idx)
		if ok {
			t.Errorf("Get(%d) reported ok, want false", idx)
		}
		if v != 0 {
			t.Errorf("Get(%d) = %d, want the zero value", idx, v)
		}
	}
}

func TestSliceDelete(t *testing.T) {
	s := NewSlice[int](0)
	s.Append(1)
	s.Append(2)
	s.Append(3)

	s.Delete(1)
	if got := s.Slice(); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("Slice() after Delete(1) = %v, want [1 3]", got)
	}

	s.Delete(-1)
	s.Delete(99)
	if s.Len() != 2 {
		t.Errorf("Len() after out-of-range deletes = %d, want 2", s.Len())
	}

	s.Delete(0)
	s.Delete(0)
	if s.Len() != 0 {
		t.Errorf("Len() after deleting everything = %d, want 0", s.Len())
	}
}

func TestSliceClear(t *testing.T) {
	s := NewSlice[int](0)
	s.Append(1)
	s.Append(2)

	before := s.Cap()
	s.Clear()
	if s.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", s.Len())
	}
	if s.Cap() < before {
		t.Errorf("Clear() should retain capacity: %d < %d", s.Cap(), before)
	}

	s.Append(9)
	if got := s.Slice(); !slices.Equal(got, []int{9}) {
		t.Errorf("Slice() after Clear+Append = %v, want [9]", got)
	}
}

func TestSliceClone(t *testing.T) {
	s := NewSlice[int](0)
	s.Append(1)
	s.Append(2)

	clone := s.Clone()
	if !slices.Equal(clone.Slice(), []int{1, 2}) {
		t.Errorf("Clone() = %v", clone.Slice())
	}
	clone.Append(3)
	if s.Len() != 2 {
		t.Errorf("mutating the clone changed the original: %v", s.Slice())
	}

	out := s.Slice()
	out[0] = 99
	if v, _ := s.Get(0); v != 1 {
		t.Errorf("mutating the returned slice changed the source: %d", v)
	}
}

func TestSliceIterator(t *testing.T) {
	s := NewSlice[int](0)
	for i := 1; i <= 3; i++ {
		s.Append(i)
	}

	var got []int
	for v := range s.Iterator() {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("Iterator() = %v, want [1 2 3]", got)
	}

	count := 0
	for range s.Iterator() {
		count++
		break
	}
	if count != 1 {
		t.Errorf("early break yielded %d entries, want 1", count)
	}
}

func TestSliceString(t *testing.T) {
	var empty Slice[int]
	if got := empty.String(); got != "[]" {
		t.Errorf("String() on a zero value = %q, want []", got)
	}

	s := NewSlice[int](0)
	s.Append(1)
	s.Append(2)
	if got := s.String(); got != "[1 2]" {
		t.Errorf("String() = %q, want [1 2]", got)
	}
}

func TestSliceJSONRoundTrip(t *testing.T) {
	s := NewSlice[int](0)
	s.Append(1)
	s.Append(2)

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(data) != "[1,2]" {
		t.Errorf("Marshal() = %s, want [1,2]", data)
	}

	var got Slice[int]
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !slices.Equal(got.Slice(), []int{1, 2}) {
		t.Errorf("after round trip = %v", got.Slice())
	}

	var zero Slice[int]
	data, err = json.Marshal(&zero)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("Marshal() of a zero value = %s, want []", data)
	}
}

func TestSliceConcurrentAppends(t *testing.T) {
	const (
		workers = 16
		perEach = 100
	)

	s := NewSlice[int](workers * perEach)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perEach; i++ {
				s.Append(i)
			}
		}()
	}
	wg.Wait()

	if s.Len() != workers*perEach {
		t.Errorf("Len() = %d, want %d", s.Len(), workers*perEach)
	}

	total := 0
	for v := range s.Iterator() {
		total += v
	}
	wantTotal := 0
	for i := 0; i < perEach; i++ {
		wantTotal += i * workers
	}
	if total != wantTotal {
		t.Errorf("sum = %d, want %d (updates were lost)", total, wantTotal)
	}
}
