// Code generated by "stringer -type=Packages"; DO NOT EDIT.

package dep

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[UnknownPkg-0]
	_ = x[DebianBased-1]
	_ = x[REMBased-2]
}

const _Packages_name = "UnknownPkgDebianBasedREMBased"

var _Packages_index = [...]uint8{0, 10, 21, 29}

func (i Packages) String() string {
	if i < 0 || i >= Packages(len(_Packages_index)-1) {
		return "Packages(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Packages_name[_Packages_index[i]:_Packages_index[i+1]]
}
