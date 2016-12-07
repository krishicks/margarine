package margarine

import (
	"fmt"
	"go/ast"
	"go/token"
	"unicode"
	"unicode/utf8"
)

type FakifyOpts struct {
	StructName string
}

func Fakify(genDecl *ast.GenDecl, funcDecls *[]*ast.FuncDecl) {
	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		panic("specs no good!")
	}

	typeSpec.Name.Name = "Fake" + typeSpec.Name.Name

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		panic("typeSpec type no good!")
	}

	for _, funcDecl := range *funcDecls {
		stubFuncOnStruct(structType, funcDecl)

		privateName := privatize(funcDecl.Name.Name)
		addMutexForFuncOnStruct(structType, privateName)

		if funcDecl.Type.Params.NumFields() > 0 {
			addArgsForCallForFuncOnStruct(structType, funcDecl, privateName)
		}

		if funcDecl.Type.Results.NumFields() > 0 {
			addReturnsStructField(structType, funcDecl, privateName)
		}
	}

	addInvocationsMethod(funcDecls, typeSpec.Name.Name)
	addRecordInvocationMethod(funcDecls, typeSpec.Name.Name)

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("invocations")},
		Type: &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: ast.NewIdent("[][]interface{}"),
		},
	})

	structType.Fields.List = append(structType.Fields.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("invocationsMutex")},
		Type: &ast.SelectorExpr{
			X:   ast.NewIdent("sync"),
			Sel: ast.NewIdent("RWMutex"),
		},
	})
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
				Names: []*ast.Ident{
					ast.NewIdent(fmt.Sprintf("arg%d", i)),
				},
				Type: fieldType,
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

func addRecordInvocationMethod(funcDecls *[]*ast.FuncDecl, structName string) {
	newFuncDecls := append(*funcDecls, &ast.FuncDecl{
		Name: ast.NewIdent("recordInvocation"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("key")},
						Type:  ast.NewIdent("string"),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("args")},
						Type:  ast.NewIdent("[]interface{}"),
					},
				},
			},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("fake")},
					Type:  &ast.StarExpr{X: ast.NewIdent(structName)},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("fake"),
								Sel: ast.NewIdent("invocationsMutex"),
							},
							Sel: ast.NewIdent("Lock"),
						},
					},
				},
				&ast.DeferStmt{
					Call: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("fake"),
								Sel: ast.NewIdent("invocationsMutex"),
							},
							Sel: ast.NewIdent("Unlock"),
						},
					},
				},
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X: &ast.SelectorExpr{
							X:   ast.NewIdent("fake"),
							Sel: ast.NewIdent("invocations"),
						},
						Op: token.EQL,
						Y: &ast.BasicLit{
							Kind:  token.STRING,
							Value: "nil",
						},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Tok: token.ASSIGN,
								Lhs: []ast.Expr{
									&ast.SelectorExpr{
										X:   ast.NewIdent("fake"),
										Sel: ast.NewIdent("invocations"),
									},
								},
								Rhs: []ast.Expr{ast.NewIdent("map[string][][]interface{}{}")},
							},
						},
					},
				},
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X: &ast.IndexExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("fake"),
								Sel: ast.NewIdent("invocations"),
							},
							Index: ast.NewIdent("key"),
						},
						Op: token.EQL,
						Y: &ast.BasicLit{
							Kind:  token.STRING,
							Value: "nil",
						},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Tok: token.ASSIGN,
								Lhs: []ast.Expr{
									&ast.IndexExpr{
										X: &ast.SelectorExpr{
											X:   ast.NewIdent("fake"),
											Sel: ast.NewIdent("invocations"),
										},
										Index: ast.NewIdent("key"),
									},
								},
								Rhs: []ast.Expr{ast.NewIdent("[][]interface{}{}")},
							},
						},
					},
				},

				&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{
						&ast.IndexExpr{
							X: &ast.SelectorExpr{
								X:   ast.NewIdent("fake"),
								Sel: ast.NewIdent("invocations"),
							},
							Index: ast.NewIdent("key"),
						},
					},
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: ast.NewIdent("append"),
							Args: []ast.Expr{
								&ast.IndexExpr{
									X: &ast.SelectorExpr{
										X:   ast.NewIdent("fake"),
										Sel: ast.NewIdent("invocations"),
									},
									Index: ast.NewIdent("key"),
								},
								ast.NewIdent("args"),
							},
						},
					},
				},
			},
		},
	})

	*funcDecls = newFuncDecls
}

func addInvocationsMethod(funcDecls *[]*ast.FuncDecl, structName string) {
	statements := []ast.Stmt{
		// fake.invocationsMutex.Lock()
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.SelectorExpr{
						X:   ast.NewIdent("fake"),
						Sel: ast.NewIdent("invocationsMutex"),
					},
					Sel: ast.NewIdent("RLock"),
				},
			},
		},
		// defer fake.invocationsMutext.RUnlock()
		&ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.SelectorExpr{
						X:   ast.NewIdent("fake"),
						Sel: ast.NewIdent("invocationsMutex"),
					},
					Sel: ast.NewIdent("RUnlock"),
				},
			},
		},
	}

	for _, funcDecl := range *funcDecls {
		methodMutexFieldName := privatize(funcDecl.Name.Name) + "Mutex"

		statements = append(statements,
			// fake.methodMutex.RLock()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.SelectorExpr{
							X:   ast.NewIdent("fake"),
							Sel: ast.NewIdent(methodMutexFieldName),
						},
						Sel: ast.NewIdent("RLock"),
					},
				},
			},
			// defer fake.methodMutex.RUnlock()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.SelectorExpr{
							X:   ast.NewIdent("fake"),
							Sel: ast.NewIdent(methodMutexFieldName),
						},
						Sel: ast.NewIdent("RUnlock"),
					},
				},
			},
		)
	}

	statements = append(statements, &ast.ReturnStmt{
		// return fake.invocations
		Results: []ast.Expr{
			&ast.SelectorExpr{
				X:   ast.NewIdent("fake"),
				Sel: ast.NewIdent("invocations"),
			},
		},
	})

	*funcDecls = append(*funcDecls, &ast.FuncDecl{
		Name: ast.NewIdent("Invocations"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{{
					Type: ast.NewIdent("map[string][][]interface{}"),
				}},
			},
		},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("fake")},
					Type:  &ast.StarExpr{X: ast.NewIdent(structName)},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: statements,
		},
	})
}
