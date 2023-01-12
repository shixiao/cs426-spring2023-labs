package string_set

type StringSet interface {
	// Add string s to the StringSet and return whether the string was inserted
	// in the set.
	Add(key string) bool

	// Return the number of unique strings in the set
	Count() int

	// Return all strings matching a regex `pattern` within a range `[begin,
	// end)` lexicographically (for Part C)
	PredRange(begin string, end string, pattern string) []string
}

type LockedStringSet struct {
}

func MakeLockedStringSet() LockedStringSet {
	return LockedStringSet{}
}

func (stringSet *LockedStringSet) Add(key string) bool {
	return false
}

func (stringSet *LockedStringSet) Count() int {
	return 0
}

func (stringSet *LockedStringSet) PredRange(begin string, end string, pattern string) []string {
	return make([]string, 0)
}
