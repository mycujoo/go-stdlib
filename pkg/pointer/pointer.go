package pointer

// To converts from value to the pointer
func To[V any](v V) *V {
	return &v
}

// Optional returns the pointer to the value if the value is not zero,
// otherwise it returns nil
func Optional[V comparable](v V) *V {
	var zero V
	if v == zero {
		return nil
	}
	return &v
}

// From extracts value from the pointer if not nil,
// otherwise it returns zero value
func From[V any](v *V) V {
	if v == nil {
		var res V
		return res
	}
	return *v
}
