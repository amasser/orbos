// Code generated by goderive DO NOT EDIT.

package hostname

// deriveFilter returns a list of all items in the list that matches the predicate.
func deriveFilter(predicate func(string) bool, list []string) []string {
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
