package margarine

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type StructifyOpts struct {
	InterfaceName      string
	RecvName           string
	StructName         string
	PackageName        string
	ParseMode          parser.Mode
	PreserveParamNames bool
}

func Structify(src []byte, opts StructifyOpts) (*ast.File, error) {
	if opts.InterfaceName == "" {
		return nil, errors.New("must provide interface name")
	}

	structName := "someStruct"
	if opts.StructName != "" {
		structName = opts.StructName
	}

	recvName := string(structName[0])
	if opts.RecvName != "" {
		recvName = opts.RecvName
	}

	parseMode := parser.AllErrors
	if opts.ParseMode != 0 {
		parseMode = opts.ParseMode
	}

	packageName := "mypackage"
	if opts.PackageName != "" {
		packageName = opts.PackageName
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, parseMode)
	if err != nil {
		return nil, err
	}

	var obj *ast.Object
	ast.Inspect(f, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		switch v := node.(type) {
		case *ast.TypeSpec:
			if v.Name != nil && v.Name.Name != opts.InterfaceName {
				return true
			}

			obj = v.Name.Obj
			return false
		default:
			return true
		}
	})

	if obj == nil {
		return nil, errors.New("could not find interface")
	}

	// structObj not required when not using pointer receiver
	structObj := ast.NewObj(ast.Typ, structName)
	recvObj := ast.NewObj(ast.Typ, recvName)

	decls := []ast.Decl{
		&ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: &ast.Ident{
						Name: structObj.Name,
						Obj:  structObj,
					},
					Type: &ast.StructType{
						Fields: &ast.FieldList{
							List: nil,
						},
					},
				},
			},
		},
	}

	funcDecls := getFuncDecls(structObj, recvObj, obj, []ast.Decl{}, opts.PreserveParamNames)
	for i := range funcDecls {
		decls = append(decls, funcDecls[i])
	}

	f = &ast.File{
		Name: &ast.Ident{
			Name: packageName,
		},
		Decls: decls,
	}

	return f, nil
}

func getFuncDecls(
	structObj, recvObj, o *ast.Object,
	funcDecls []ast.Decl,
	preserveParamNames bool,
) []ast.Decl {
	newFuncDecls := []ast.Decl{}

	typeSpec, ok := o.Decl.(*ast.TypeSpec)
	if !ok {
		panic("not ok")
	}

	typ, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		panic("not ok")
	}

	for _, field := range typ.Methods.List {
		switch v := field.Type.(type) {
		case *ast.FuncType:
			newFuncDecls = append(newFuncDecls, newFuncDecl(
				recvObj,
				structObj,
				field.Names[0].Name,
				v.Params,
				v.Results,
				preserveParamNames,
			))
		case *ast.Ident:
			if v.Obj == nil {
				panic("found embedded interface with no associated interface")
			}

			newFuncDecls = append(newFuncDecls, getFuncDecls(structObj, recvObj, v.Obj, funcDecls, preserveParamNames)...)
		default:
			fmt.Printf("%#v\n", v)
			continue
		}
	}

	return newFuncDecls
}

func newFuncDecl(
	recvObj, structObj *ast.Object,
	funcName string,
	params, results *ast.FieldList,
	preserveParamNames bool,
) ast.Decl {
	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: funcName},
		Type: &ast.FuncType{},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{
							Name: recvObj.Name,
							Obj:  recvObj,
						},
					},
					// TODO: support non-pointer receiver
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: structObj.Name,
							Obj:  structObj,
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			Lbrace: token.NoPos,
			List:   nil,
			Rbrace: token.NoPos,
		},
	}

	if params != nil {
		fl := &ast.FieldList{}
		var i int
		for _, field := range params.List {
			if len(field.Names) > 0 {
				for _, ident := range field.Names {
					i++
					if preserveParamNames {
						fl.List = append(fl.List, &ast.Field{
							Names: []*ast.Ident{ast.NewIdent(ident.Name)},
							Type:  field.Type,
						})
					} else {
						fl.List = append(fl.List, &ast.Field{
							Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
							Type:  field.Type,
						})
					}
				}
			} else {
				i++
				fl.List = append(fl.List, &ast.Field{
					Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
					Type:  field.Type,
				})
			}
		}

		funcDecl.Type.Params = fl
	}

	if results != nil {
		fl := &ast.FieldList{}
		for _, field := range results.List {
			if len(field.Names) > 0 {
				for range field.Names {
					fl.List = append(fl.List, &ast.Field{
						Type: field.Type,
					})
				}
			}
		}

		funcDecl.Type.Results = fl
	}

	return funcDecl
}
