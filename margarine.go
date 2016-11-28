package margarine

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	"golang.org/x/tools/imports"
)

type StructifyOpts struct {
	InterfaceName string
	RecvName      string
	StructName    string
	PackageName   string
	ParseMode     parser.Mode
}

func Structify(src []byte, opts StructifyOpts) ([]byte, error) {
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

	structObj := ast.NewObj(ast.Typ, structName)
	recvObj := ast.NewObj(ast.Typ, recvName)

	genDecl := &ast.GenDecl{
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
	}

	funcDecls := getFuncDecls(structObj, recvObj, obj, []ast.Decl{})

	decls := []ast.Decl{
		genDecl,
	}

	for i := range funcDecls {
		decls = append(decls, funcDecls[i])
	}

	f = &ast.File{
		Name: &ast.Ident{
			Name: packageName,
		},
		Decls: decls,
	}

	fset = token.NewFileSet()

	var buf bytes.Buffer
	err = format.Node(&buf, fset, f)
	if err != nil {
		panic(err)
	}

	out, err := imports.Process("src.go", buf.Bytes(), nil)
	if err != nil {
		panic(err)
	}

	return out, nil
}

func getFuncDecls(structObj, recvObj, o *ast.Object, funcDecls []ast.Decl) []ast.Decl {
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
			funcDecl := &ast.FuncDecl{
				Recv: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{
								{
									Name: recvObj.Name,
									Obj:  recvObj,
								},
							},
							Type: &ast.StarExpr{
								X: &ast.Ident{
									Name: structObj.Name,
									Obj:  structObj,
								},
							},
						},
					},
				},
				Name: &ast.Ident{
					Name: field.Names[0].Name,
					Obj:  nil,
				},
				Type: &ast.FuncType{
					Params:  v.Params,
					Results: v.Results,
				},
				Body: &ast.BlockStmt{
					List: nil,
				},
			}

			newFuncDecls = append(newFuncDecls, funcDecl)
		case *ast.Ident:
			if v.Obj == nil {
				panic("found embedded interface with no associated interface")
			}

			newFuncDecls = append(newFuncDecls, getFuncDecls(structObj, recvObj, v.Obj, funcDecls)...)
		default:
			fmt.Printf("%#v\n", v)
			continue
		}
	}

	return newFuncDecls
}
