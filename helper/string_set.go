package helper

type StringSet struct {
	m map[string]struct{}
}

func NewStringSet(ss ...string) *StringSet {
	m := map[string]struct{}{}

	for i := range ss {
		m[ss[i]] = struct{}{}
	}

	return &StringSet{
		m: m,
	}
}

func (s *StringSet) Add(k string) {
	s.m[k] = struct{}{}
}

func (s *StringSet) Remove(k string) {
	delete(s.m, k)
}

func (s StringSet) Has(k string) bool {
	_, ok := s.m[k]
	return ok
}

func (s *StringSet) Empty() bool {
	return len(s.m) == 0
}
