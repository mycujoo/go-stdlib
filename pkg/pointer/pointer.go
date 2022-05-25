package pointer

// From returns pointer to the value
func From[V any](v V) *V {
	return &v
}

// Optional returns the pointer to the value if the value is not zero,
// otherwise it returns nil
func Optional[V comparable](v V) *V {
	// If the passed value has `IsZero() bool` method (for example, time.Time instance),
	// it is used to determine if the value is zero.
	if z, ok := any(v).(interface{ IsZero() bool }); ok {
		if z.IsZero() {
			return nil
		}
		return &v
	}

	var zero V
	if v == zero {
		return nil
	}
	return &v
}

// Unwrap extracts value from the pointer if not nil,
// otherwise it returns zero value
func Unwrap[V any](v *V) V {
	if v == nil {
		var res V
		return res
	}
	return *v
}
