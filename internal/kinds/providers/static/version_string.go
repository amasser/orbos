// Code generated by "stringer -type Version"; DO NOT EDIT.

package static

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[unknown-0]
	_ = x[v1-1]
}

const _Version_name = "unknownv1"

var _Version_index = [...]uint8{0, 7, 9}

func (i Version) String() string {
	if i < 0 || i >= Version(len(_Version_index)-1) {
		return "Version(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Version_name[_Version_index[i]:_Version_index[i+1]]
}
