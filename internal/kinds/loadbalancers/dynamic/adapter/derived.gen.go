// Code generated by goderive DO NOT EDIT.

package adapter

import (
	infra "github.com/caos/orbiter/internal/kinds/clusters/core/infra"
	model "github.com/caos/orbiter/internal/kinds/loadbalancers/dynamic/model"
)

// deriveFilter returns a list of all items in the list that matches the predicate.
func deriveFilter(predicate func(infra.Compute) bool, list []infra.Compute) []infra.Compute {
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

// deriveFmapSourceVRRPHealthChecks returns a list where each element of the input list has been morphed by the input function.
func deriveFmapSourceVRRPHealthChecks(f func(model.Source) string, list []model.Source) []string {
	out := make([]string, len(list))
	for i, elem := range list {
		out[i] = f(elem)
	}
	return out
}

// deriveAny reports whether the predicate returns true for any of the elements in the given slice.
func deriveAny(pred func(model.Source) bool, list []model.Source) bool {
	for _, elem := range list {
		if pred(elem) {
			return true
		}
	}
	return false
}
