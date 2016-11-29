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
	var structType *ast.StructType
	var obj *ast.Object
	ast.Inspect(file, func(node ast.Node) bool {
		if v, ok := node.(*ast.TypeSpec); ok {
			if v.Name == nil || v.Name.Name != opts.StructName {
				return true
			}

			if structType, ok = v.Type.(*ast.StructType); ok {
				obj = v.Name.Obj
				return false
			}
		}
		return true
	})

	if structType == nil {
		panic("could not find struct")
	}

	if obj == nil {
		panic("no obj for typespec")
	}

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

		if funcDecl.Type.Results.NumFields() > 0 {
			addReturnsStructField(structType, funcDecl, privateName)
		}
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

func addReturnsStructField(structType *ast.StructType, funcDecl *ast.FuncDecl, privateName string) {
	var fields []*ast.Field
	var i int
	for _, field := range funcDecl.Type.Results.List {
		i++
		fields = append(fields, &ast.Field{
			Type:  field.Type,
			Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("result%d", i))},
		})
	}

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(privateName + "Returns")},
		Type: &ast.StructType{
			Fields: &ast.FieldList{
				List: fields,
			},
		},
	})
}

func addArgsForCallForFuncOnStruct(structType *ast.StructType, funcDecl *ast.FuncDecl, privateName string) {
	var fields []*ast.Field
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

			fields = append(fields, &ast.Field{
				Type:  fieldType,
				Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
			})
		}
	}

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(privateName + "ArgsForCall")},
		Type: &ast.ArrayType{
			Elt: &ast.StructType{
				Fields: &ast.FieldList{
					List: fields,
				},
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
	var singularizeFields = func(fl *ast.FieldList) *ast.FieldList {
		result := &ast.FieldList{}

		if fl.NumFields() > 0 {
			for _, field := range fl.List {
				var i int
				for range field.Names {
					i++
					result.List = append(result.List, &ast.Field{
						Type: field.Type,
					})
				}
			}
		}

		return result
	}

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{
			ast.NewIdent(funcDecl.Name.Name + "Stub"), // missing Obj; necessary? would include Kind: var
		},
		Type: &ast.FuncType{
			Params:  singularizeFields(funcDecl.Type.Params),
			Results: singularizeFields(funcDecl.Type.Results),
		},
	})
}
