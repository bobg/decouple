package decouple

import "go/types"

func getChanType(typ types.Type) *types.Chan {
	switch typ := typ.(type) {
	case *types.Chan:
		return typ
	case *types.Named:
		return getChanType(typ.Underlying())
	default:
		return nil
	}
}

func getSig(typ types.Type) *types.Signature {
	switch typ := typ.(type) {
	case *types.Signature:
		return typ
	case *types.Named:
		return getSig(typ.Underlying())
	default:
		return nil
	}
}

func getInterface(typ types.Type) *types.Interface {
	switch typ := typ.(type) {
	case *types.Interface:
		return typ
	case *types.Named:
		return getInterface(typ.Underlying())
	default:
		return nil
	}
}

func getMap(typ types.Type) *types.Map {
	switch typ := typ.(type) {
	case *types.Map:
		return typ
	case *types.Named:
		return getMap(typ.Underlying())
	default:
		return nil
	}
}
