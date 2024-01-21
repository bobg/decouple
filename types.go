package decouple

import "go/types"

func getType[T types.Type](typ types.Type) T {
	switch typ := typ.(type) {
	case T:
		return typ
	case *types.Named:
		return getType[T](typ.Underlying())
	default:
		return zero[T]()
	}
}

// Returns the zero value for any type.
func zero[T any]() (res T) {
	return
}
