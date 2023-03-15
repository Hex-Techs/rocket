package tools

type EmptyType struct{}

var empty EmptyType

func New[T comparable]() Set[T] {
	return Set[T]{
		m: make(map[T]EmptyType),
	}
}

// Set struct
type Set[T comparable] struct {
	m map[T]EmptyType
}

func (s *Set[T]) Insert(val T) {
	s.m[val] = empty
}

func (s *Set[T]) Remove(val T) {
	delete(s.m, val)
}

func (s *Set[T]) Has(val T) bool {
	_, ok := s.m[val]
	return ok
}

func (s *Set[T]) Len() int {
	return len(s.m)
}

func (s *Set[T]) Equals(other Set[T]) bool {
	if s.Len() != other.Len() {
		return false
	}
	for k := range s.m {
		if _, ok := other.m[k]; !ok {
			return false
		}
	}
	return true
}

// set intersection
func (s *Set[T]) Intersection(other Set[T]) Set[T] {
	newSet := new(Set[T])
	newSet.m = make(map[T]EmptyType)
	for _, i := range other.Items() {
		if s.Has(i) {
			newSet.Insert(i)
		}
	}
	return *newSet
}

// set union
func (s *Set[T]) Union(other Set[T]) Set[T] {
	newSet := new(Set[T])
	newSet.m = make(map[T]EmptyType)
	for _, i := range s.Items() {
		newSet.Insert(i)
	}
	for _, i := range other.Items() {
		if !newSet.Has(i) {
			newSet.Insert(i)
		}
	}
	return *newSet
}

// set difference
func (s *Set[T]) Difference(other Set[T]) Set[T] {
	newSet := new(Set[T])
	newSet.m = make(map[T]EmptyType)
	for _, i := range s.Items() {
		if !other.Has(i) {
			newSet.Insert(i)
		}
	}
	return *newSet
}

func (s *Set[T]) Clear() {
	s.m = make(map[T]EmptyType)
}

func (s *Set[T]) Items() []T {
	item := make([]T, 0)
	for k := range s.m {
		item = append(item, k)
	}
	return item
}
