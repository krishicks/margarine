package margarine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/imports"
)

type FakifyOpts struct {
	StructName string
}

func Fakify(file *ast.File, opts FakifyOpts) *ast.File {
	// add stubs to struct
	var structType *ast.StructType
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		var obj *ast.Object
		switch v := node.(type) {
		case *ast.TypeSpec:
			if v.Name != nil && v.Name.Name != opts.StructName {
				return true
			}

			var ok bool
			structType, ok = v.Type.(*ast.StructType)
			if !ok {
				return true
			}

			obj = v.Name.Obj
		case *ast.FuncDecl:
			if structType == nil {
				break
			}

			if obj != nil { // hmm
				if expr, ok := v.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := expr.X.(*ast.Ident); ok {
						if ident.Obj != obj {
							// this isn't the object you're looking for
							break
						}
					}
				}
			}

			params := &ast.FieldList{}
			for _, field := range v.Type.Params.List {
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
			if v.Type.Results != nil {
				for _, field := range v.Type.Results.List {
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
					ast.NewIdent(v.Name.Name + "Stub"), // missing Obj; necessary? would include Kind: var
				},
				Type: &ast.FuncType{
					Params:  params,
					Results: results,
				},
			})
		default:
			// fmt.Printf("%p -> %#v\n", v, v)
		}

		return true
	})

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

func fakeMethod() {
}
