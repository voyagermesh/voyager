package reflect

import r "reflect"

// https://stackoverflow.com/a/23555352/244009
func IsZero(i interface{}) bool {
	v := r.ValueOf(i)
	switch v.Kind() {
	case r.Func, r.Map, r.Slice, r.Ptr:
		return v.IsNil()
	case r.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && IsZero(v.Index(i))
		}
		return z
	case r.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && IsZero(v.Field(i))
		}
		return z
	}
	// Compare other types directly:
	z := r.Zero(v.Type())
	return v.Interface() == z.Interface()
}
