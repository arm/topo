package collections

import (
	"maps"
	"slices"
)

type Set[T comparable] struct {
	elements map[T]struct{}
}

func NewSet[T comparable](items ...T) Set[T] {
	s := Set[T]{elements: make(map[T]struct{})}
	for _, item := range items {
		s.Add(item)
	}
	return s
}

func (s *Set[T]) Add(item T) {
	s.elements[item] = struct{}{}
}

func (s *Set[T]) ToSlice() []T {
	return slices.Collect(maps.Keys(s.elements))
}

func (s *Set[T]) Contains(item T) bool {
	_, exists := s.elements[item]
	return exists
}
