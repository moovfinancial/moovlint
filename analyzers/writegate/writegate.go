package writegate

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:      "writegate",
	Doc:       "require observability/sql writes to run in an events consumer handler, not an HTTP handler",
	Run:       run,
	FactTypes: []analysis.Fact{new(dbWriteFact)},
}

type dbWriteFact struct{}

func (*dbWriteFact) AFact() {}

func (*dbWriteFact) String() string { return "writes DB" }

var sqlWriteMethods = map[string]bool{
	"Exec":                 true,
	"ExecContext":          true,
	"ExecContextRetryable": true,
	"InTxScope":            true,
}

var eventingHandlerNames = map[string]bool{
	"EventHandlerContext":       true,
	"RecordHandler":             true,
	"RawMessageHandler":         true,
	"EventMessageHandler":       true,
	"EventHeaderMessageHandler": true,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) && !strings.HasPrefix(pass.Pkg.Path(), "testdata/") {
		return nil, nil
	}

	localWrites := map[*types.Func]bool{}
	decls := map[*types.Func]*ast.FuncDecl{}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Name == nil {
				continue
			}
			obj, _ := pass.TypesInfo.Defs[fn.Name].(*types.Func)
			if obj == nil {
				continue
			}
			decls[obj] = fn
			if bodyWrites(pass, fn.Body, nil) {
				localWrites[obj] = true
			}
		}
	}

	changed := true
	for changed {
		changed = false
		for obj, fn := range decls {
			if localWrites[obj] {
				continue
			}
			if bodyWrites(pass, fn.Body, localWrites) {
				localWrites[obj] = true
				changed = true
			}
		}
		before := len(localWrites)
		markInterfaceMethods(pass, localWrites)
		if len(localWrites) > before {
			changed = true
		}
	}

	for obj, writes := range localWrites {
		if writes {
			pass.ExportObjectFact(obj, &dbWriteFact{})
		}
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		reportHTTPWrites(pass, file, localWrites)
	}
	return nil, nil
}

func markInterfaceMethods(pass *analysis.Pass, local map[*types.Func]bool) {
	scope := pass.Pkg.Scope()
	var ifaces []*types.Named
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		if _, ok := tn.Type().Underlying().(*types.Interface); ok {
			if n, ok := tn.Type().(*types.Named); ok {
				ifaces = append(ifaces, n)
			}
		}
	}
	for fn, writes := range local {
		if !writes {
			continue
		}
		sig, _ := fn.Type().(*types.Signature)
		if sig == nil || sig.Recv() == nil {
			continue
		}
		recv := sig.Recv().Type()
		for _, ifaceNamed := range ifaces {
			iface := ifaceNamed.Underlying().(*types.Interface)
			if !types.Implements(recv, iface) {
				continue
			}
			obj, _, _ := types.LookupFieldOrMethod(ifaceNamed, false, pass.Pkg, fn.Name())
			if im, ok := obj.(*types.Func); ok {
				local[im] = true
			}
		}
	}
}

func bodyWrites(pass *analysis.Pass, body *ast.BlockStmt, local map[*types.Func]bool) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isSQLWriteCall(pass, call) {
			found = true
			return false
		}
		if callee := calleeFunc(pass, call); callee != nil && funcHasWrite(pass, callee, local) {
			found = true
			return false
		}
		return true
	})
	return found
}

func funcHasWrite(pass *analysis.Pass, fn *types.Func, local map[*types.Func]bool) bool {
	if fn == nil {
		return false
	}
	if local != nil && local[fn] {
		return true
	}
	var fact dbWriteFact
	return pass.ImportObjectFact(fn, &fact)
}

func reportHTTPWrites(pass *analysis.Pass, file *ast.File, local map[*types.Func]bool) {
	var stack []frame
	var inspect func(n ast.Node) bool
	inspect = func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch x := n.(type) {
		case *ast.FuncDecl:
			stack = append(stack, frameFrom(pass, x.Name.Name, x.Type, pass.TypesInfo.Defs[x.Name]))
			if x.Body != nil {
				ast.Inspect(x.Body, inspect)
			}
			stack = stack[:len(stack)-1]
			return false
		case *ast.FuncLit:
			stack = append(stack, frameFromLit(pass, x))
			if x.Body != nil {
				ast.Inspect(x.Body, inspect)
			}
			stack = stack[:len(stack)-1]
			return false
		case *ast.CallExpr:
			if !inHTTP(stack) || inEvent(stack) {
				return true
			}
			name := callName(x)
			if isSQLWriteCall(pass, x) || funcHasWrite(pass, calleeFunc(pass, x), local) {
				pass.Report(analysis.Diagnostic{
					Pos:     x.Pos(),
					Message: fmt.Sprintf("database write %s must run in an events consumer handler; HTTP handlers should produce an event instead", name),
				})
			}
		}
		return true
	}
	ast.Inspect(file, inspect)
}

type frame struct {
	http  bool
	event bool
}

func frameFromLit(pass *analysis.Pass, lit *ast.FuncLit) frame {
	fr := frame{}
	if t := pass.TypesInfo.TypeOf(lit); t != nil {
		if isEventingHandlerType(t) {
			fr.event = true
		}
		if sig := asSig(t); sig != nil {
			if signatureIsHTTP(pass, sig) {
				fr.http = true
			}
			if signatureLooksLikeEventHandler(sig) {
				fr.event = true
			}
		}
	} else if funcTypeHasResponseWriter(lit.Type) {
		fr.http = true
	}
	return fr
}

func frameFrom(pass *analysis.Pass, name string, ft *ast.FuncType, obj types.Object) frame {
	fr := frame{}
	var sig *types.Signature
	if fn, ok := obj.(*types.Func); ok {
		sig, _ = fn.Type().(*types.Signature)
	}
	if sig == nil && ft != nil {
		if t := pass.TypesInfo.TypeOf(ft); t != nil {
			sig = asSig(t)
		}
	}
	if sig != nil {
		if signatureIsHTTP(pass, sig) {
			fr.http = true
		}
		if signatureLooksLikeEventHandler(sig) {
			fr.event = true
		}
		if signatureReturnsEventHandler(sig) {
			fr.event = true
		}
		if isEventingHandlerType(sig) {
			fr.event = true
		}
	} else if funcTypeHasResponseWriter(ft) {
		fr.http = true
	}
	if eventingHandlerNames[name] {
		fr.event = true
	}
	return fr
}

func inHTTP(stack []frame) bool {
	for _, fr := range stack {
		if fr.http {
			return true
		}
	}
	return false
}

func inEvent(stack []frame) bool {
	for _, fr := range stack {
		if fr.event {
			return true
		}
	}
	return false
}

func isSQLWriteCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn := calleeFunc(pass, call)
	if fn == nil {
		return false
	}
	if !sqlWriteMethods[fn.Name()] {
		return false
	}
	pkg := fn.Pkg()
	return pkg != nil && isObsSQLPackage(pkg.Path())
}

func calleeFunc(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	var id *ast.Ident
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		id = fun
	case *ast.SelectorExpr:
		id = fun.Sel
	default:
		return nil
	}
	fn, _ := pass.TypesInfo.ObjectOf(id).(*types.Func)
	return fn
}

func callName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return fun.Sel.Name
	}
	return "call"
}

func isMoovPath(path, suffix string) bool {
	return strings.Contains(path, "github.com/moovfinancial/") && strings.HasSuffix(path, suffix)
}

func isObsSQLPackage(path string) bool {
	return isMoovPath(path, "/observability/sql")
}

func isEventingPackage(path string) bool {
	return isMoovPath(path, "/events/go/eventing")
}

func isEventsPackage(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return strings.Contains(pkg.Path(), "github.com/moovfinancial/events/")
}

func isEventingHandlerType(t types.Type) bool {
	t = types.Unalias(t)
	named, ok := t.(*types.Named)
	if !ok {
		return signatureLooksLikeEventHandler(asSig(t))
	}
	obj := named.Obj()
	if obj != nil && obj.Pkg() != nil && isEventingPackage(obj.Pkg().Path()) && eventingHandlerNames[obj.Name()] {
		return true
	}
	return signatureLooksLikeEventHandler(asSig(named.Underlying()))
}

func asSig(t types.Type) *types.Signature {
	if t == nil {
		return nil
	}
	sig, _ := t.Underlying().(*types.Signature)
	return sig
}

func signatureReturnsEventHandler(sig *types.Signature) bool {
	if sig == nil || sig.Results().Len() != 1 {
		return false
	}
	return isEventingHandlerType(sig.Results().At(0).Type())
}

func signatureLooksLikeEventHandler(sig *types.Signature) bool {
	if sig == nil || sig.Params().Len() < 2 || sig.Results().Len() != 1 {
		return false
	}
	if !isErrorType(sig.Results().At(0).Type()) {
		return false
	}
	if !isNamed(sig.Params().At(0).Type(), "context", "Context") {
		return false
	}
	p1 := sig.Params().At(1).Type()
	p1 = types.Unalias(p1)
	if ptr, ok := p1.(*types.Pointer); ok {
		if n := namedOf(ptr.Elem()); n != nil && n.Obj().Name() == "Event" && isEventsPackage(n.Obj().Pkg()) {
			return true
		}
	}
	if sl, ok := p1.(*types.Slice); ok {
		elem := sl.Elem()
		if ptr, ok := elem.(*types.Pointer); ok {
			elem = ptr.Elem()
		}
		n := namedOf(elem)
		if n == nil || n.Obj() == nil {
			return false
		}
		switch n.Obj().Name() {
		case "EventMessage", "RawMessage", "EventHeadersMessage", "Record":
			return true
		}
	}
	return false
}

func signatureIsHTTP(pass *analysis.Pass, sig *types.Signature) bool {
	if sig == nil {
		return false
	}
	for i := 0; i < sig.Params().Len(); i++ {
		if isResponseWriter(pass, sig.Params().At(i).Type()) {
			return true
		}
	}
	return false
}

func funcTypeHasResponseWriter(ft *ast.FuncType) bool {
	if ft == nil || ft.Params == nil {
		return false
	}
	for _, field := range ft.Params.List {
		switch t := field.Type.(type) {
		case *ast.SelectorExpr:
			if ident, ok := t.X.(*ast.Ident); ok && ident.Name == "http" && t.Sel.Name == "ResponseWriter" {
				return true
			}
		case *ast.Ident:
			if t.Name == "ResponseWriter" {
				return true
			}
		}
	}
	return false
}

func isResponseWriter(pass *analysis.Pass, t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	rw := lookupPkgType(pass, "net/http", "ResponseWriter")
	if rw == nil {
		n := namedOf(t)
		return n != nil && n.Obj() != nil && n.Obj().Pkg() != nil &&
			n.Obj().Pkg().Path() == "net/http" && n.Obj().Name() == "ResponseWriter"
	}
	iface, ok := rw.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	if types.Identical(t, rw) {
		return true
	}
	return types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface)
}

func lookupPkgType(pass *analysis.Pass, pkgPath, name string) types.Type {
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() != pkgPath {
			continue
		}
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			return nil
		}
		return obj.Type()
	}
	return nil
}

func isNamed(t types.Type, pkgSuffix, name string) bool {
	n := namedOf(t)
	if n == nil || n.Obj() == nil {
		return false
	}
	if n.Obj().Name() != name {
		return false
	}
	if pkgSuffix == "" {
		return true
	}
	if n.Obj().Pkg() == nil {
		return false
	}
	return n.Obj().Pkg().Path() == pkgSuffix || strings.HasSuffix(n.Obj().Pkg().Path(), "/"+pkgSuffix)
}

func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	errObj := types.Universe.Lookup("error")
	if errObj == nil {
		return false
	}
	errT := errObj.Type()
	if types.Identical(t, errT) {
		return true
	}
	iface, ok := errT.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	return types.Implements(t, iface)
}

func namedOf(t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	t = types.Unalias(t)
	n, _ := t.(*types.Named)
	return n
}
