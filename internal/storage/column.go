package storage

// Column is the untyped view of a TypedColumn[T]. Archetype code works through
// it; generic fast paths downcast with col.(*TypedColumn[T]).
//
// Values passed as any are T values, not *T. A value of any other type panics
// with the runtime's type-assertion error; callers validate types beforehand.
type Column interface {
	// Len returns the number of rows.
	Len() int
	// AppendZero appends a zero value row.
	AppendZero()
	// AppendAny appends v, which must be a T.
	AppendAny(v any)
	// SetAny overwrites row with v, which must be a T. row must be in range.
	SetAny(row int, v any)
	// PtrAny returns &data[row] as a *T boxed in any. It doesn't allocate:
	// a pointer fits in the interface word. The pointer is valid until the
	// next append or swap-remove on this column.
	PtrAny(row int) any
	// SwapRemove removes row by moving the last row into it and shrinking the
	// column. row must be in range.
	SwapRemove(row int)
	// MoveRowTo appends a copy of row to dst, which must be a column of the
	// same T. It doesn't remove row; call SwapRemove for that.
	MoveRowTo(dst Column, row int)
}

// TypedColumn stores the values of one component type, one per archetype row.
//
// Storage is a plain []T rather than raw bytes: components may hold pointers,
// strings and slices, and only a typed slice lets the GC see them.
type TypedColumn[T any] struct {
	data []T
}

// NewTypedColumn returns an empty column of T. It is the natural body of a
// column factory passed to NewIndex.
func NewTypedColumn[T any]() *TypedColumn[T] { return &TypedColumn[T]{} }

// Len returns the number of rows.
func (c *TypedColumn[T]) Len() int { return len(c.data) }

// AppendZero appends a zero value row.
func (c *TypedColumn[T]) AppendZero() {
	var zero T
	c.data = append(c.data, zero)
}

// Append appends v.
func (c *TypedColumn[T]) Append(v T) { c.data = append(c.data, v) }

// AppendAny appends v, which must be a T.
func (c *TypedColumn[T]) AppendAny(v any) { c.data = append(c.data, v.(T)) }

// Set overwrites row with v. It panics if row is out of range.
func (c *TypedColumn[T]) Set(row int, v T) { c.data[row] = v }

// SetAny overwrites row with v, which must be a T.
func (c *TypedColumn[T]) SetAny(row int, v any) { c.data[row] = v.(T) }

// Ptr returns a pointer to row. It stays valid until the next append or
// swap-remove on this column. It panics if row is out of range.
func (c *TypedColumn[T]) Ptr(row int) *T { return &c.data[row] }

// PtrAny returns Ptr(row) boxed in any, without allocating.
func (c *TypedColumn[T]) PtrAny(row int) any { return &c.data[row] }

// SwapRemove removes row by moving the last row into it.
func (c *TypedColumn[T]) SwapRemove(row int) {
	last := len(c.data) - 1
	c.data[row] = c.data[last]
	// Zero the vacated slot: it stays inside the backing array and would
	// otherwise keep whatever its pointers, strings and slices reference alive.
	var zero T
	c.data[last] = zero
	c.data = c.data[:last]
}

// MoveRowTo appends a copy of row to dst, which must be a *TypedColumn[T].
func (c *TypedColumn[T]) MoveRowTo(dst Column, row int) {
	d := dst.(*TypedColumn[T])
	d.data = append(d.data, c.data[row])
}
