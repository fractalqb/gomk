// Code generated by "stringer -type BuildHint"; DO NOT EDIT.

package bees

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Build-1]
	_ = x[RootNode-2]
	_ = x[DepChanged-3]
}

const _BuildHint_name = "BuildRootNodeDepChanged"

var _BuildHint_index = [...]uint8{0, 5, 13, 23}

func (i BuildHint) String() string {
	i -= 1
	if i < 0 || i >= BuildHint(len(_BuildHint_index)-1) {
		return "BuildHint(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _BuildHint_name[_BuildHint_index[i]:_BuildHint_index[i+1]]
}