package decouple

import "go/types"

func getType[T types.Type](typ types.Type) T {
	switch typ := typ.(type) {
	case T:
		return typ
	case *types.Named:
		return getType[T](typ.Underlying())
	default:
		var t T
		return t
	}
}
