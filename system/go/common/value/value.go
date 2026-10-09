package value

// Nullable returns the value pointed to by p, or nil (database NULL) when p is nil.
func Nullable[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
