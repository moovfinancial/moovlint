package placement

import "go/types"

// Model selects exported data structs rather than service implementations.
func Model(t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok || !named.Obj().Exported() {
		return nil
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok || st.NumFields() == 0 {
		return nil
	}
	for i := 0; i < named.NumMethods(); i++ {
		if named.Method(i).Name() != "Validate" {
			return nil
		}
	}
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Exported() || field.Embedded() {
			return nil
		}
		switch field.Type().Underlying().(type) {
		case *types.Signature, *types.Interface, *types.Chan:
			return nil
		}
	}
	return named
}
