// Package parser extracts and classifies wire.Build()
// arguments from injector functions.
package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Injector represents a wire injector function.
type Injector struct {
	FuncName string
	Filename string
	Line     int
	BuildCall *ast.CallExpr
}

// ClassifiedArgs holds the categorized wire.Build() args.
type ClassifiedArgs struct {
	ProviderSets        []*ProviderSetRef
	StandaloneProviders []*StandaloneProviderRef
	StructProviders     []*StructProvider
	Bindings            []*BindingRef
}

// ProviderSetRef is a reference to a pkg.ProviderSet var.
type ProviderSetRef struct {
	VarName  string
	PkgPath  string
	PkgAlias string
	Obj      *types.Var
	Line     int
}

// QualifiedName returns "alias.VarName".
func (p *ProviderSetRef) QualifiedName() string {
	if p.PkgAlias != "" {
		return p.PkgAlias + "." + p.VarName
	}
	return p.VarName
}

// StandaloneProviderRef is a function used directly
// in wire.Build().
type StandaloneProviderRef struct {
	FuncName string
	PkgPath  string
	Obj      types.Object
	Line     int
}

// StructProvider represents wire.Struct(new(T), ...).
type StructProvider struct {
	TargetType types.Type
	Fields     []string // empty = all ("*")
	Line       int
}

// BindingRef represents wire.Bind(new(I), new(*C)).
type BindingRef struct {
	Iface    types.Type
	Concrete types.Type
	Line     int
}

// FindInjectors finds all wire.Build() calls in pkg.
func FindInjectors(
	pkg *packages.Package,
) ([]*Injector, error) {
	var injectors []*Injector

	for _, file := range pkg.Syntax {
		fname := pkg.Fset.File(file.Pos()).Name()
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			call := findBuildCall(pkg, fd.Body)
			if call == nil {
				continue
			}
			pos := pkg.Fset.Position(call.Pos())
			injectors = append(injectors, &Injector{
				FuncName:  fd.Name.Name,
				Filename:  fname,
				Line:      pos.Line,
				BuildCall: call,
			})
		}
	}
	return injectors, nil
}

// findBuildCall finds wire.Build(...) in a function body.
func findBuildCall(
	pkg *packages.Package,
	body *ast.BlockStmt,
) *ast.CallExpr {
	for _, stmt := range body.List {
		es, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		if isWireBuildCall(pkg, call) {
			return call
		}
	}
	return nil
}

// isWireBuildCall checks if call is wire.Build(...).
func isWireBuildCall(
	pkg *packages.Package,
	call *ast.CallExpr,
) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Build" {
		return false
	}
	obj := pkg.TypesInfo.Uses[sel.Sel]
	if obj == nil {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	return isWirePackage(fn.Pkg())
}

func isWirePackage(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	path := pkg.Path()
	return path == "github.com/google/wire" ||
		path == "github.com/goforj/wire"
}

// ClassifyArgs categorizes each wire.Build() argument.
func ClassifyArgs(
	pkg *packages.Package,
	inj *Injector,
) (*ClassifiedArgs, error) {
	result := &ClassifiedArgs{}

	for _, arg := range inj.BuildCall.Args {
		pos := pkg.Fset.Position(arg.Pos())

		switch expr := arg.(type) {
		case *ast.SelectorExpr:
			if err := classifySelector(
				pkg, expr, pos, result,
			); err != nil {
				return nil, err
			}

		case *ast.Ident:
			classifyIdent(pkg, expr, pos, result)

		case *ast.CallExpr:
			if err := classifyCall(
				pkg, expr, pos, result,
			); err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf(
				"unexpected arg type %T at %s",
				arg, pos)
		}
	}

	return result, nil
}

func classifySelector(
	pkg *packages.Package,
	sel *ast.SelectorExpr,
	pos token.Position,
	result *ClassifiedArgs,
) error {
	obj := pkg.TypesInfo.Uses[sel.Sel]
	if obj == nil {
		return fmt.Errorf(
			"unresolved selector %s at %s",
			sel.Sel.Name, pos)
	}

	switch o := obj.(type) {
	case *types.Var:
		alias := ""
		if id, ok := sel.X.(*ast.Ident); ok {
			alias = id.Name
		}
		result.ProviderSets = append(
			result.ProviderSets,
			&ProviderSetRef{
				VarName:  sel.Sel.Name,
				PkgPath:  o.Pkg().Path(),
				PkgAlias: alias,
				Obj:      o,
				Line:     pos.Line,
			},
		)
	case *types.Func:
		result.StandaloneProviders = append(
			result.StandaloneProviders,
			&StandaloneProviderRef{
				FuncName: sel.Sel.Name,
				PkgPath:  o.Pkg().Path(),
				Obj:      o,
				Line:     pos.Line,
			},
		)
	default:
		return fmt.Errorf(
			"unexpected object type %T for %s",
			obj, sel.Sel.Name)
	}

	return nil
}

func classifyIdent(
	pkg *packages.Package,
	id *ast.Ident,
	pos token.Position,
	result *ClassifiedArgs,
) {
	obj := pkg.TypesInfo.Uses[id]
	if obj == nil {
		obj = pkg.TypesInfo.Defs[id]
	}
	if obj == nil {
		return
	}

	switch o := obj.(type) {
	case *types.Func:
		result.StandaloneProviders = append(
			result.StandaloneProviders,
			&StandaloneProviderRef{
				FuncName: id.Name,
				PkgPath:  pkg.PkgPath,
				Obj:      o,
				Line:     pos.Line,
			},
		)
	case *types.Var:
		result.ProviderSets = append(
			result.ProviderSets,
			&ProviderSetRef{
				VarName: id.Name,
				PkgPath: pkg.PkgPath,
				Obj:     o,
				Line:    pos.Line,
			},
		)
	}
}

func classifyCall(
	pkg *packages.Package,
	call *ast.CallExpr,
	pos token.Position,
	result *ClassifiedArgs,
) error {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	obj := pkg.TypesInfo.Uses[sel.Sel]
	if obj == nil {
		return nil
	}
	fn, ok := obj.(*types.Func)
	if !ok || !isWirePackage(fn.Pkg()) {
		return nil
	}

	switch sel.Sel.Name {
	case "Struct":
		sp, err := parseWireStruct(pkg, call, pos)
		if err != nil {
			return err
		}
		result.StructProviders = append(
			result.StructProviders, sp)

	case "Bind":
		br, err := parseWireBind(pkg, call, pos)
		if err != nil {
			return err
		}
		result.Bindings = append(result.Bindings, br)
	}

	return nil
}

// parseWireStruct extracts the type and fields from
// wire.Struct(new(T), "*") or wire.Struct(new(T), "F1", "F2").
func parseWireStruct(
	pkg *packages.Package,
	call *ast.CallExpr,
	pos token.Position,
) (*StructProvider, error) {
	if len(call.Args) < 2 {
		return nil, fmt.Errorf(
			"wire.Struct needs >= 2 args at %s", pos)
	}

	typ, err := extractNewType(pkg, call.Args[0])
	if err != nil {
		return nil, fmt.Errorf(
			"wire.Struct first arg at %s: %w", pos, err)
	}

	sp := &StructProvider{
		TargetType: typ,
		Line:       pos.Line,
	}

	for _, arg := range call.Args[1:] {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		val := lit.Value[1 : len(lit.Value)-1]
		sp.Fields = append(sp.Fields, val)
	}

	return sp, nil
}

// parseWireBind extracts interface and concrete types from
// wire.Bind(new(I), new(*C)).
func parseWireBind(
	pkg *packages.Package,
	call *ast.CallExpr,
	pos token.Position,
) (*BindingRef, error) {
	if len(call.Args) < 2 {
		return nil, fmt.Errorf(
			"wire.Bind needs 2 args at %s", pos)
	}

	iface, err := extractNewType(pkg, call.Args[0])
	if err != nil {
		return nil, fmt.Errorf(
			"wire.Bind iface at %s: %w", pos, err)
	}

	concrete, err := extractNewType(pkg, call.Args[1])
	if err != nil {
		return nil, fmt.Errorf(
			"wire.Bind concrete at %s: %w", pos, err)
	}

	return &BindingRef{
		Iface:    iface,
		Concrete: concrete,
		Line:     pos.Line,
	}, nil
}

// extractNewType gets the T from new(T).
func extractNewType(
	pkg *packages.Package,
	expr ast.Expr,
) (types.Type, error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("expected new() call")
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok || id.Name != "new" {
		return nil, fmt.Errorf("expected new() call")
	}
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("new() takes 1 arg")
	}

	typ := pkg.TypesInfo.TypeOf(call.Args[0])
	if typ == nil {
		return nil, fmt.Errorf("cannot resolve type")
	}

	return typ, nil
}
