package data_structure

type Set struct {
	members map[string]struct{}
}

func NewSet() *Set {
	return &Set{
		members: make(map[string]struct{}),
	}
}

func (s *Set) Add(members ...string) int {
	added := 0
	for _, member := range members {
		_, exist := s.members[member]
		if !exist {
			s.members[member] = struct{}{}
			added++
		}
	}
	return added
}

func (s *Set) Remove(members ...string) int {
	removed := 0
	for _, member := range members {
		_, exist := s.members[member]
		if exist {
			delete(s.members, member)
			removed++
		}
	}
	return removed
}

func (s *Set) Members() []string {
	members := make([]string, 0, len(s.members))
	for member := range s.members {
		members = append(members, member)
	}
	return members
}

func (set *Set) IsMember(s string) bool {
	_, exist := set.members[s]
	return exist
}
