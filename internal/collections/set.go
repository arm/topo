package collections

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
	slice := make([]T, 0, len(s.elements))
	for item := range s.elements {
		slice = append(slice, item)
	}
	return slice
}

func (s *Set[T]) Contains(item T) bool {
	_, exists := s.elements[item]
	return exists
}
