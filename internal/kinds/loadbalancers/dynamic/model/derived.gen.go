// Code generated by goderive DO NOT EDIT.

package model

// deriveUnique returns a list containing only the unique items from the input list.
// It does this by reusing the input list.
func deriveUnique(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	return deriveKeys(deriveSet(list))
}

// deriveFilter returns a list of all items in the list that matches the predicate.
func deriveFilter(predicate func(Source) bool, list []Source) []Source {
	j := 0
	for i, elem := range list {
		if predicate(elem) {
			if i != j {
				list[j] = list[i]
			}
			j++
		}
	}
	return list[:j]
}

// deriveKeys returns the keys of the input map as a slice.
func deriveKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// deriveSet returns the input list as a map with the items of the list as the keys of the map.
func deriveSet(list []string) map[string]struct{} {
	set := make(map[string]struct{}, len(list))
	for _, v := range list {
		set[v] = struct{}{}
	}
	return set
}
