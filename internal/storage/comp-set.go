package storage

import "slices"

// IsSet reports whether ids is strictly ascending, i.e. sorted and free of
// duplicates. Component sets are always kept in this form.
func IsSet(ids []CompID) bool {
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			return false
		}
	}
	return true
}

// Contains reports whether the set contains id. Cost is O(log n).
func Contains(set []CompID, id CompID) bool {
	_, ok := slices.BinarySearch(set, id)
	return ok
}

// Union appends the union of the sets a and b to dst[:0] and returns it. It
// doesn't allocate when dst has enough capacity. dst must not alias a or b.
func Union(dst, a, b []CompID) []CompID {
	dst = dst[:0]
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] < b[j]:
			dst = append(dst, a[i])
			i++
		case a[i] > b[j]:
			dst = append(dst, b[j])
			j++
		default:
			dst = append(dst, a[i])
			i++
			j++
		}
	}
	dst = append(dst, a[i:]...)
	return append(dst, b[j:]...)
}

// Diff appends the elements of a that are not in b to dst[:0] and returns it.
// It doesn't allocate when dst has enough capacity. dst must not alias a or b.
func Diff(dst, a, b []CompID) []CompID {
	dst = dst[:0]
	j := 0
	for _, id := range a {
		for j < len(b) && b[j] < id {
			j++
		}
		if j < len(b) && b[j] == id {
			continue
		}
		dst = append(dst, id)
	}
	return dst
}
