package decouple

import (
	"fmt"
	"go/types"
	"testing"
)

type typeConstraint interface {
	types.Type
	comparable
}

func TestGetType(t *testing.T) {
	cases := []struct {
		t                            types.Type
		isChan, isSig, isIntf, isMap bool
	}{{
		t: types.NewStruct(nil, nil),
	}, {
		t:      types.NewChan(types.SendRecv, types.NewStruct(nil, nil)),
		isChan: true,
	}, {
		t:     types.NewSignatureType(nil, nil, nil, nil, nil, false),
		isSig: true,
	}, {
		t:      types.NewInterfaceType(nil, nil),
		isIntf: true,
	}, {
		t:     types.NewMap(types.NewStruct(nil, nil), types.NewStruct(nil, nil)),
		isMap: true,
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			checkType[*types.Chan](t, tc.t, tc.isChan)
			checkType[*types.Chan](t, types.NewNamed(types.NewTypeName(0, nil, "foo", nil), tc.t, nil), tc.isChan)

			checkType[*types.Signature](t, tc.t, tc.isSig)
			checkType[*types.Signature](t, types.NewNamed(types.NewTypeName(0, nil, "foo", nil), tc.t, nil), tc.isSig)

			checkType[*types.Interface](t, tc.t, tc.isIntf)
			checkType[*types.Interface](t, types.NewNamed(types.NewTypeName(0, nil, "foo", nil), tc.t, nil), tc.isIntf)

			checkType[*types.Map](t, tc.t, tc.isMap)
			checkType[*types.Map](t, types.NewNamed(types.NewTypeName(0, nil, "foo", nil), tc.t, nil), tc.isMap)
		})
	}
}

func checkType[T typeConstraint](t *testing.T, inp types.Type, isType bool) {
	t.Helper()

	var zero T

	got := getType[T](inp)
	if (got != zero) != isType {
		t.Errorf("is-type[%T] is %v, want %v", zero, got != zero, isType)
	}
}
