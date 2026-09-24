package writegate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:      "writegate",
		Doc:       "advisory: require observability SQL writes to run in an events consumer; HTTP and gRPC API handlers produce an event that both active regions consume",
		FactTypes: []analysis.Fact{new(dbWriteFact)},
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory writegate checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
}

type dbWriteFact struct{}

func (*dbWriteFact) AFact() {}

func (*dbWriteFact) String() string { return "writes DB" }

var sqlWriteMethods = map[string]bool{
	"Exec":                 true,
	"ExecContext":          true,
	"ExecContextRetryable": true,
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

	funcValues := map[*types.Var]*types.Func{}
	varWrites := map[*types.Var]bool{}
	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		collectFuncValues(pass, file, funcValues, varWrites, nil)
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
			obj = originFunc(obj)
			decls[obj] = fn
			if bodyWrites(pass, fn.Body, nil, funcValues, varWrites) {
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
			if bodyWrites(pass, fn.Body, localWrites, funcValues, varWrites) {
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

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		collectFuncValues(pass, file, funcValues, varWrites, localWrites)
	}

	for obj, writes := range localWrites {
		if writes && obj.Pkg() == pass.Pkg {
			pass.ExportObjectFact(obj, &dbWriteFact{})
		}
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		reportHTTPWrites(pass, file, localWrites, funcValues, varWrites)
	}
	return nil, nil
}

func collectFuncValues(pass *analysis.Pass, file *ast.File, funcValues map[*types.Var]*types.Func, varWrites map[*types.Var]bool, local map[*types.Func]bool) {
	add := func(lhs ast.Expr, rhs ast.Expr) {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			return
		}
		v, ok := pass.TypesInfo.ObjectOf(ident).(*types.Var)
		if !ok && pass.TypesInfo.Defs[ident] != nil {
			v, _ = pass.TypesInfo.Defs[ident].(*types.Var)
		}
		if v == nil {
			return
		}
		if fn := exprFunc(pass, rhs); fn != nil {
			funcValues[v] = originFunc(fn)
		}
		if lit, ok := rhs.(*ast.FuncLit); ok && lit.Body != nil && bodyWrites(pass, lit.Body, local, funcValues, varWrites) {
			varWrites[v] = true
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) != len(x.Rhs) {
				return true
			}
			for i := range x.Lhs {
				add(x.Lhs[i], x.Rhs[i])
			}
		case *ast.ValueSpec:
			for i, name := range x.Names {
				if i >= len(x.Values) {
					break
				}
				add(name, x.Values[i])
			}
		}
		return true
	})
}

func exprFunc(pass *analysis.Pass, expr ast.Expr) *types.Func {
	expr = unwrapIndex(expr)
	switch e := expr.(type) {
	case *ast.Ident:
		fn, _ := pass.TypesInfo.ObjectOf(e).(*types.Func)
		return originFunc(fn)
	case *ast.SelectorExpr:
		fn, _ := pass.TypesInfo.ObjectOf(e.Sel).(*types.Func)
		return originFunc(fn)
	}
	return nil
}

// markInterfaceMethods copies a concrete method's write mark onto matching
// interface methods in this package. One writing implementer marks the
// interface method, so every call through that interface is treated as a
// write. That over-reports event-only implementations; that is the safe
// direction for this gate. Facts are still exported only for this package.
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

func bodyWrites(pass *analysis.Pass, body *ast.BlockStmt, local map[*types.Func]bool, funcValues map[*types.Var]*types.Func, varWrites map[*types.Var]bool) bool {
	if body == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isSQLWriteCall(pass, call) {
			found = true
			return false
		}
		if isInTxScopeCall(pass, call) && inTxScopeWrites(pass, call, local, funcValues, varWrites) {
			found = true
			return false
		}
		if callWrites(pass, call, local, funcValues, varWrites) {
			found = true
			return false
		}
		return true
	})
	return found
}

func inTxScopeWrites(pass *analysis.Pass, call *ast.CallExpr, local map[*types.Func]bool, funcValues map[*types.Var]*types.Func, varWrites map[*types.Var]bool) bool {
	for _, arg := range call.Args {
		switch a := arg.(type) {
		case *ast.FuncLit:
			if a.Body != nil && bodyWrites(pass, a.Body, local, funcValues, varWrites) {
				return true
			}
		default:
			if fn := exprFunc(pass, a); fn != nil && funcHasWrite(pass, fn, local) {
				return true
			}
			if v := exprVar(pass, a); v != nil && (varWrites[v] || funcHasWrite(pass, funcValues[v], local)) {
				return true
			}
		}
	}
	return false
}

func callWrites(pass *analysis.Pass, call *ast.CallExpr, local map[*types.Func]bool, funcValues map[*types.Var]*types.Func, varWrites map[*types.Var]bool) bool {
	if fn := calleeFunc(pass, call); fn != nil && funcHasWrite(pass, fn, local) {
		return true
	}
	if v := calleeVar(pass, call); v != nil {
		if varWrites[v] {
			return true
		}
		if fn := funcValues[v]; fn != nil && funcHasWrite(pass, fn, local) {
			return true
		}
	}
	return false
}

func funcHasWrite(pass *analysis.Pass, fn *types.Func, local map[*types.Func]bool) bool {
	fn = originFunc(fn)
	if fn == nil {
		return false
	}
	if local != nil && local[fn] {
		return true
	}
	var fact dbWriteFact
	return pass.ImportObjectFact(fn, &fact)
}

func reportHTTPWrites(pass *analysis.Pass, file *ast.File, local map[*types.Func]bool, funcValues map[*types.Var]*types.Func, varWrites map[*types.Var]bool) {
	var stack []frame
	reported := map[token.Pos]bool{}
	var inspect func(n ast.Node) bool
	inspect = func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch x := n.(type) {
		case *ast.FuncDecl:
			stack = append(stack, frameFrom(pass, x.Type, pass.TypesInfo.Defs[x.Name]))
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
			if inAPI(stack) && !inEvent(stack) {
				name := callName(x)
				if isSQLWriteCall(pass, x) || callWrites(pass, x, local, funcValues, varWrites) {
					if !reported[x.Pos()] {
						reported[x.Pos()] = true
						pass.Report(analysis.Diagnostic{
							Pos:     x.Pos(),
							Message: fmt.Sprintf("database write %s must run in an events consumer handler; API handlers should produce an event so both active regions apply the write", name),
						})
					}
				}
			}
			for _, arg := range x.Args {
				lit, ok := arg.(*ast.FuncLit)
				if !ok || lit.Body == nil {
					ast.Inspect(arg, inspect)
					continue
				}
				fr := frameFromLit(pass, lit)
				if !fr.api && !fr.event && inAPI(stack) {
					fr.api = true
				}
				stack = append(stack, fr)
				ast.Inspect(lit.Body, inspect)
				stack = stack[:len(stack)-1]
			}
			return false
		}
		return true
	}
	ast.Inspect(file, inspect)
}

type frame struct {
	api   bool
	event bool
}

func frameFromLit(pass *analysis.Pass, lit *ast.FuncLit) frame {
	fr := frame{}
	if t := pass.TypesInfo.TypeOf(lit); t != nil {
		applySig(&fr, pass, t)
	} else if funcTypeHasResponseWriter(lit.Type) {
		fr.api = true
	}
	return fr
}

func frameFrom(pass *analysis.Pass, ft *ast.FuncType, obj types.Object) frame {
	fr := frame{}
	var sig *types.Signature
	if fn, ok := obj.(*types.Func); ok {
		sig, _ = originFunc(fn).Type().(*types.Signature)
	}
	if sig == nil && ft != nil {
		if t := pass.TypesInfo.TypeOf(ft); t != nil {
			sig = asSig(t)
		}
	}
	if sig != nil {
		applySig(&fr, pass, sig)
		if sig.Recv() != nil && embedsUnimplementedServer(sig.Recv().Type()) {
			fr.api = true
		}
	} else if funcTypeHasResponseWriter(ft) {
		fr.api = true
	}
	return fr
}

func applySig(fr *frame, pass *analysis.Pass, t types.Type) {
	if isEventingHandlerType(t) {
		fr.event = true
	}
	sig := asSig(t)
	if sig == nil {
		return
	}
	if signatureIsHTTP(pass, sig) {
		fr.api = true
	}
	if signatureLooksLikeEventHandler(sig) {
		fr.event = true
	}
	if signatureReturnsEventHandler(sig) {
		fr.event = true
	}
}

func inAPI(stack []frame) bool {
	return len(stack) > 0 && stack[len(stack)-1].api
}

func embedsUnimplementedServer(t types.Type) bool {
	seen := map[types.Type]bool{}
	var walk func(types.Type) bool
	walk = func(t types.Type) bool {
		if t == nil {
			return false
		}
		t = deref(types.Unalias(t))
		if seen[t] {
			return false
		}
		seen[t] = true
		n := namedOf(t)
		if n != nil && n.Obj() != nil {
			name := n.Obj().Name()
			if strings.HasPrefix(name, "Unimplemented") && strings.HasSuffix(name, "Server") {
				return true
			}
			t = n.Underlying()
		}
		st, ok := t.(*types.Struct)
		if !ok {
			return false
		}
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if f.Embedded() && walk(f.Type()) {
				return true
			}
		}
		return false
	}
	return walk(t)
}

func inEvent(stack []frame) bool {
	return len(stack) > 0 && stack[len(stack)-1].event
}

func isSQLWriteCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn := calleeFunc(pass, call)
	if fn == nil || !sqlWriteMethods[fn.Name()] {
		return false
	}
	pkg := fn.Pkg()
	return pkg != nil && isObsSQLPackage(pkg.Path())
}

func isInTxScopeCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn := calleeFunc(pass, call)
	if fn == nil || fn.Name() != "InTxScope" {
		return false
	}
	pkg := fn.Pkg()
	return pkg != nil && isObsSQLPackage(pkg.Path())
}

func unwrapIndex(expr ast.Expr) ast.Expr {
	for {
		switch e := expr.(type) {
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		default:
			return expr
		}
	}
}

func calleeFunc(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	return exprFunc(pass, call.Fun)
}

func calleeVar(pass *analysis.Pass, call *ast.CallExpr) *types.Var {
	return exprVar(pass, call.Fun)
}

func exprVar(pass *analysis.Pass, expr ast.Expr) *types.Var {
	expr = unwrapIndex(expr)
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}
	v, _ := pass.TypesInfo.ObjectOf(ident).(*types.Var)
	return v
}

func originFunc(fn *types.Func) *types.Func {
	if fn == nil {
		return nil
	}
	if o := fn.Origin(); o != nil {
		return o
	}
	return fn
}

func callName(call *ast.CallExpr) string {
	fun := unwrapIndex(call.Fun)
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return "call"
}

func isMoovPath(path, suffix string) bool {
	return strings.Contains(path, "github.com/moovfinancial/") && strings.HasSuffix(path, suffix)
}

func isObsSQLPackage(path string) bool {
	if !strings.Contains(path, "github.com/moovfinancial/") {
		return false
	}
	if strings.HasSuffix(path, "/observability/sql") {
		return true
	}
	return strings.Contains(path, "observability") && strings.HasSuffix(path, "/pkg/sql")
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

func isFranzRecord(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return strings.Contains(pkg.Path(), "github.com/twmb/franz-go")
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
	p1 := types.Unalias(sig.Params().At(1).Type())
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
		if n == nil || n.Obj() == nil || n.Obj().Pkg() == nil {
			return false
		}
		switch n.Obj().Name() {
		case "EventMessage", "RawMessage", "EventHeadersMessage":
			return isEventingPackage(n.Obj().Pkg().Path())
		case "Record":
			return isEventingPackage(n.Obj().Pkg().Path()) || isFranzRecord(n.Obj().Pkg())
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
		rw = responseWriterFromType(t)
	}
	if rw == nil {
		n := namedOf(deref(t))
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

func responseWriterFromType(t types.Type) types.Type {
	n := namedOf(deref(t))
	if n == nil || n.Obj() == nil || n.Obj().Pkg() == nil {
		return nil
	}
	pkg := n.Obj().Pkg()
	if pkg.Path() == "net/http" {
		if obj := pkg.Scope().Lookup("ResponseWriter"); obj != nil {
			return obj.Type()
		}
	}
	for _, im := range pkg.Imports() {
		if im.Path() == "net/http" {
			if obj := im.Scope().Lookup("ResponseWriter"); obj != nil {
				return obj.Type()
			}
		}
	}
	return nil
}

func lookupPkgType(pass *analysis.Pass, pkgPath, name string) types.Type {
	seen := map[string]bool{}
	var walk func(*types.Package) types.Type
	walk = func(pkg *types.Package) types.Type {
		if pkg == nil || seen[pkg.Path()] {
			return nil
		}
		seen[pkg.Path()] = true
		if pkg.Path() == pkgPath {
			if obj := pkg.Scope().Lookup(name); obj != nil {
				return obj.Type()
			}
		}
		for _, im := range pkg.Imports() {
			if t := walk(im); t != nil {
				return t
			}
		}
		return nil
	}
	return walk(pass.Pkg)
}

func deref(t types.Type) types.Type {
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		return types.Unalias(p.Elem())
	}
	return t
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
