package margarine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/imports"
)

type FakifyOpts struct {
	StructName string
}

func Fakify(file *ast.File, opts FakifyOpts) *ast.File {
	// find struct
	var structType *ast.StructType
	var obj *ast.Object
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		v, ok := node.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if v.Name != nil && v.Name.Name != opts.StructName {
			return true
		}

		structType, ok = v.Type.(*ast.StructType)
		if !ok {
			return true
		}

		obj = v.Name.Obj

		return false
	})

	if structType == nil {
		panic("could not find struct")
	}

	if obj == nil {
		panic("no obj for typespec")
	}

	// find methods for struct
	var funcDecls []*ast.FuncDecl
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.FuncDecl); ok {
			switch expr := v.Recv.List[0].Type.(type) {
			//TODO: support non-pointer receiver
			case *ast.StarExpr:
				if ident, ok := expr.X.(*ast.Ident); ok {
					if ident.Obj == obj {
						funcDecls = append(funcDecls, v)
					}
				}
			}
		}

		return true
	})

	for _, funcDecl := range funcDecls {
		stubFuncOnStruct(structType, funcDecl)

		privateName := privatize(funcDecl.Name.Name)
		addMutexForFuncOnStruct(structType, privateName)
		addArgsForCallForFuncOnStruct(structType, funcDecl, privateName)
	}

	fset := token.NewFileSet()
	var buf bytes.Buffer
	err := format.Node(&buf, fset, file)
	if err != nil {
		panic(err)
	}

	out, err := imports.Process("src.go", buf.Bytes(), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	return nil
}

func privatize(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

func addArgsForCallForFuncOnStruct(structType *ast.StructType, funcDecl *ast.FuncDecl, privateName string) {
	var elts []*ast.Field
	var i int
	for _, field := range funcDecl.Type.Params.List {
		for range field.Names {
			i++
			var fieldType ast.Expr
			if ellipsis, ok := field.Type.(*ast.Ellipsis); ok {
				fieldType = &ast.ArrayType{Elt: ellipsis.Elt}
			} else {
				fieldType = field.Type
			}

			elts = append(elts, &ast.Field{
				Type:  fieldType,
				Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
			})
		}
	}

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(privateName + "ArgsForCall")},
		Type: &ast.ArrayType{
			Elt: &ast.StructType{
				Fields: &ast.FieldList{List: elts},
			},
		},
	})
}

func addMutexForFuncOnStruct(structType *ast.StructType, privateName string) {
	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Type: &ast.SelectorExpr{
			X:   ast.NewIdent("sync"),
			Sel: ast.NewIdent("RWMutex"),
		},
		Names: []*ast.Ident{ast.NewIdent(privateName + "Mutex")},
	})
}

func stubFuncOnStruct(structType *ast.StructType, funcDecl *ast.FuncDecl) {
	params := &ast.FieldList{}
	for _, field := range funcDecl.Type.Params.List {
		var i int
		if len(field.Names) > 0 {
			// this path isn't hit if Structify is used; it removes multi-named fields
			for range field.Names {
				i++
				params.List = append(params.List, &ast.Field{
					Type: field.Type,
				})
			}
		} else {
			i++
			params.List = append(params.List, &ast.Field{
				Type: field.Type,
			})
		}
	}

	results := &ast.FieldList{}
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			var i int
			if len(field.Names) > 0 {
				// this path isn't hit if Structify is used; it removes named fields
				for range field.Names {
					i++
					results.List = append(results.List, &ast.Field{
						Type: field.Type,
					})
				}
			} else {
				i++
				results.List = append(results.List, &ast.Field{
					Type: field.Type,
				})
			}
		}
	}

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{
			ast.NewIdent(funcDecl.Name.Name + "Stub"), // missing Obj; necessary? would include Kind: var
		},
		Type: &ast.FuncType{
			Params:  params,
			Results: results,
		},
	})
}
