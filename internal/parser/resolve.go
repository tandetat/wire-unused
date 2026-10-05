package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// ResolvedProvider is a provider function with its type
// signature extracted.
type ResolvedProvider struct {
	Name    string
	PkgPath string
	Inputs  []types.Type
	Outputs []types.Type
	// SetRef links back to the ProviderSet that contains
	// this provider (nil for standalone providers).
	SetRef *ProviderSetRef
	// Standalone links back to the wire.Build() arg for
	// standalone providers (nil for set providers).
	Standalone *StandaloneProviderRef
}

// ResolvedArgs holds all resolved information needed
// to build the type graph.
type ResolvedArgs struct {
	Providers []*ResolvedProvider
	Bindings  []*BindingRef
	Targets   []*StructProvider
	// SetProviders maps each ProviderSetRef to its resolved
	// providers so we can check if ALL of a set's providers
	// are unused.
	SetProviders map[*ProviderSetRef][]*ResolvedProvider
}

// ResolveAll resolves all classified args into their
// concrete provider signatures.
func ResolveAll(
	pkg *packages.Package,
	args *ClassifiedArgs,
) (*ResolvedArgs, error) {
	result := &ResolvedArgs{
		Bindings: args.Bindings,
		Targets:  args.StructProviders,
		SetProviders: make(
			map[*ProviderSetRef][]*ResolvedProvider,
		),
	}

	for _, setRef := range args.ProviderSets {
		providers, bindings, err := resolveProviderSet(
			pkg, setRef, make(map[string]bool),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve %s: %w",
				setRef.QualifiedName(), err)
		}
		result.SetProviders[setRef] = providers
		result.Providers = append(
			result.Providers, providers...)
		result.Bindings = append(
			result.Bindings, bindings...)
	}

	for _, sp := range args.StandaloneProviders {
		rp, err := resolveFunc(sp.Obj)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve %s: %w", sp.FuncName, err)
		}
		rp.Name = sp.FuncName
		rp.PkgPath = sp.PkgPath
		rp.Standalone = sp
		result.Providers = append(result.Providers, rp)
	}

	return result, nil
}

// resolveProviderSet follows a ProviderSet var to its
// wire.NewSet() definition and extracts all providers.
func resolveProviderSet(
	pkg *packages.Package,
	ref *ProviderSetRef,
	visited map[string]bool,
) ([]*ResolvedProvider, []*BindingRef, error) {
	key := ref.PkgPath + "." + ref.VarName
	if visited[key] {
		return nil, nil, nil
	}
	visited[key] = true

	defPkg := findPackage(pkg, ref.PkgPath)
	if defPkg == nil {
		return nil, nil, fmt.Errorf(
			"package %s not loaded", ref.PkgPath)
	}

	newSetCall, err := findNewSetCall(
		defPkg, ref.VarName,
	)
	if err != nil {
		return nil, nil, err
	}

	var providers []*ResolvedProvider
	var bindings []*BindingRef

	for _, arg := range newSetCall.Args {
		pos := defPkg.Fset.Position(arg.Pos())

		switch expr := arg.(type) {
		case *ast.SelectorExpr:
			obj := defPkg.TypesInfo.Uses[expr.Sel]
			if obj == nil {
				continue
			}
			switch o := obj.(type) {
			case *types.Var:
				nested := &ProviderSetRef{
					VarName: expr.Sel.Name,
					PkgPath: o.Pkg().Path(),
					Obj:     o,
				}
				np, nb, err := resolveProviderSet(
					pkg, nested, visited,
				)
				if err != nil {
					return nil, nil, err
				}
				for _, p := range np {
					p.SetRef = ref
				}
				providers = append(providers, np...)
				bindings = append(bindings, nb...)
			case *types.Func:
				rp, err := resolveFunc(o)
				if err != nil {
					continue
				}
				rp.Name = expr.Sel.Name
				rp.PkgPath = o.Pkg().Path()
				rp.SetRef = ref
				providers = append(providers, rp)
			}

		case *ast.Ident:
			obj := defPkg.TypesInfo.Uses[expr]
			if obj == nil {
				obj = defPkg.TypesInfo.Defs[expr]
			}
			if obj == nil {
				continue
			}
			switch o := obj.(type) {
			case *types.Func:
				rp, err := resolveFunc(o)
				if err != nil {
					continue
				}
				rp.Name = expr.Name
				rp.PkgPath = defPkg.PkgPath
				rp.SetRef = ref
				providers = append(providers, rp)
			case *types.Var:
				nested := &ProviderSetRef{
					VarName: expr.Name,
					PkgPath: defPkg.PkgPath,
					Obj:     o,
				}
				np, nb, err := resolveProviderSet(
					pkg, nested, visited,
				)
				if err != nil {
					return nil, nil, err
				}
				for _, p := range np {
					p.SetRef = ref
				}
				providers = append(providers, np...)
				bindings = append(bindings, nb...)
			}

		case *ast.CallExpr:
			sel, ok := expr.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			obj := defPkg.TypesInfo.Uses[sel.Sel]
			if obj == nil {
				continue
			}
			fn, ok := obj.(*types.Func)
			if !ok || !isWirePackage(fn.Pkg()) {
				continue
			}
			switch sel.Sel.Name {
			case "Bind":
				br, err := parseWireBind(
					defPkg, expr, pos,
				)
				if err != nil {
					continue
				}
				bindings = append(bindings, br)
			case "Struct":
				sp, err := parseWireStruct(
					defPkg, expr, pos,
				)
				if err != nil {
					continue
				}
				rps := resolveStructAsProvider(
					defPkg, sp,
				)
				for _, rp := range rps {
					rp.SetRef = ref
				}
				providers = append(providers, rps...)
			}
		}
	}

	return providers, bindings, nil
}

// resolveFunc extracts the input/output types from
// a function's type signature.
func resolveFunc(
	obj types.Object,
) (*ResolvedProvider, error) {
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf(
			"%s is not a function", obj.Name())
	}

	rp := &ResolvedProvider{}

	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		rp.Inputs = append(
			rp.Inputs, params.At(i).Type())
	}

	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		typ := results.At(i).Type()
		if isErrorType(typ) {
			continue
		}
		rp.Outputs = append(rp.Outputs, typ)
	}

	return rp, nil
}

// resolveStructAsProvider creates providers for a
// wire.Struct inside a ProviderSet. Each field of the
// struct that wire.Struct fills becomes an "input",
// and the struct pointer type is the "output".
func resolveStructAsProvider(
	pkg *packages.Package,
	sp *StructProvider,
) []*ResolvedProvider {
	ptrTyp := types.NewPointer(sp.TargetType)

	rp := &ResolvedProvider{
		Name:    sp.TargetType.String(),
		PkgPath: "",
		Outputs: []types.Type{ptrTyp},
	}

	underlying := sp.TargetType.Underlying()
	st, ok := underlying.(*types.Struct)
	if !ok {
		return []*ResolvedProvider{rp}
	}

	allFields := len(sp.Fields) == 1 &&
		sp.Fields[0] == "*"

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Exported() {
			continue
		}
		if allFields || containsField(
			sp.Fields, field.Name()) {
			rp.Inputs = append(
				rp.Inputs, field.Type())
		}
	}

	return []*ResolvedProvider{rp}
}

func containsField(fields []string, name string) bool {
	for _, f := range fields {
		if f == name {
			return true
		}
	}
	return false
}

// findPackage searches the loaded dependency graph for
// a package by import path.
func findPackage(
	pkg *packages.Package,
	path string,
) *packages.Package {
	if pkg.PkgPath == path {
		return pkg
	}
	visited := make(map[string]bool)
	return findPackageDFS(pkg, path, visited)
}

func findPackageDFS(
	pkg *packages.Package,
	path string,
	visited map[string]bool,
) *packages.Package {
	if visited[pkg.PkgPath] {
		return nil
	}
	visited[pkg.PkgPath] = true

	if imp, ok := pkg.Imports[path]; ok {
		return imp
	}

	for _, imp := range pkg.Imports {
		found := findPackageDFS(imp, path, visited)
		if found != nil {
			return found
		}
	}
	return nil
}

// findNewSetCall finds the wire.NewSet(...) call in
// the initializer of the named variable.
func findNewSetCall(
	pkg *packages.Package,
	varName string,
) (*ast.CallExpr, error) {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if name.Name != varName {
						continue
					}
					if i >= len(vs.Values) {
						continue
					}
					call, ok :=
						vs.Values[i].(*ast.CallExpr)
					if !ok {
						continue
					}
					if isWireNewSetCall(pkg, call) {
						return call, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf(
		"wire.NewSet not found for var %s in %s",
		varName, pkg.PkgPath)
}

func isWireNewSetCall(
	pkg *packages.Package,
	call *ast.CallExpr,
) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "NewSet" {
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

func isErrorType(typ types.Type) bool {
	return typ.String() == "error"
}
