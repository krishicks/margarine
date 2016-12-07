package margarine_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"github.com/krishicks/margarine"
	"github.com/krishicks/patrick"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Faker", func() {
	Describe("Fakify", func() {
		var (
			genDecl   *ast.GenDecl
			funcDecls []*ast.FuncDecl
		)

		BeforeEach(func() {
			src := []byte(`
package mypackage

type MyInterface interface {
	Method()
}
`)

			// this is using patrick to generate the exact input that we expect, though it
			// would be good to test that parsing a struct/methods source file also works.
			var err error
			genDecl, funcDecls, err = patrick.Pour(src, "MyInterface", "MyStruct")
			Expect(err).NotTo(HaveOccurred())

			// uncomment to test what the output ast should look like
			// osrc := []byte(`
			// package mypackage

			// import "sync"

			// type FakeStruct struct {
			// MethodStub        func()
			// methodMutex       sync.RWMutex
			// invocations      map[string][][]interface{}
			// invocationsMutex sync.RWMutex
			// }

			// func (fake *FakeStruct) Invocations() map[string][][]interface{} {
			// fake.invocationsMutex.RLock()
			// defer fake.invocationsMutex.RUnlock()
			// return fake.invocations
			// }

			// func (fake *FakeStruct) recordInvocation(key string, args []interface{}) {
			// fake.invocationsMutex.Lock()
			// defer fake.invocationsMutex.Unlock()
			// if fake.invocations == nil {
			// fake.invocations = map[string][][]interface{}{}
			// }
			// if fake.invocations[key] == nil {
			// fake.invocations[key] = [][]interface{}{}
			// }
			// fake.invocations[key] = append(fake.invocations[key], args)
			// }
			// `)

			// fset := token.NewFileSet() // positions are relative to fset
			// f, err := parser.ParseFile(fset, "src.go", osrc, parser.AllErrors)
			// if err != nil {
			// 	panic(err)
			// }
			// 			var buf bytes.Buffer
			// 			err = format.Node(&buf, fset, f)
			// 			if err != nil {
			// 				panic(err)
			// 			}
			// 			out, err := imports.Process("src.go", buf.Bytes(), nil)
			// 			if err != nil {
			// 				panic(err)
			// 			}
			// 			fmt.Println(string(out))

			// fset := token.NewFileSet() // positions are relative to fset
			// f, err := parser.ParseFile(fset, "src.go", osrc, parser.AllErrors)
			// if err != nil {
			// 	panic(err)
			// }
			// ast.Print(fset, f)
		})

		AfterEach(func() {
			fset := token.NewFileSet()

			decls := []ast.Decl{genDecl}
			for _, fd := range funcDecls {
				decls = append(decls, fd)
			}

			f := &ast.File{
				Name:  ast.NewIdent("mypackage"),
				Decls: decls,
			}

			var buf bytes.Buffer
			err := format.Node(&buf, fset, f)
			if err != nil {
				panic(err)
			}

			fmt.Println(buf.String())
		})

		JustBeforeEach(func() {
			margarine.Fakify(genDecl, &funcDecls)
		})

		It("prepends 'Fake' to the struct name", func() {
			Expect(genDecl.Specs).To(HaveLen(1))

			typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
			Expect(ok).To(BeTrue())

			Expect(typeSpec.Name).NotTo(BeNil())
			Expect(typeSpec.Name.Name).To(Equal("FakeMyStruct"))
		})

		It("adds a stub struct member for each method on the struct", func() {
			// MethodStub func()
			Expect(len(genDecl.Specs)).NotTo(BeZero())

			typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
			Expect(ok).To(BeTrue())

			structType, ok := typeSpec.Type.(*ast.StructType)
			Expect(ok).To(BeTrue())

			Expect(structType.Fields).NotTo(BeNil())
			fields := structType.Fields.List
			Expect(len(fields)).NotTo(BeZero())

			Expect(fields).To(ContainElement(&ast.Field{
				Names: []*ast.Ident{
					ast.NewIdent("MethodStub"),
				},
				Type: &ast.FuncType{
					Params:  &ast.FieldList{},
					Results: &ast.FieldList{},
				},
			}))
		})

		It("adds a mutex struct member for each method on the struct", func() {
			// methodMutex sync.RWMutex
			Expect(len(genDecl.Specs)).NotTo(BeZero())

			typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
			Expect(ok).To(BeTrue())

			structType, ok := typeSpec.Type.(*ast.StructType)
			Expect(ok).To(BeTrue())

			Expect(structType.Fields).NotTo(BeNil())
			fields := structType.Fields.List
			Expect(len(fields)).NotTo(BeZero())

			Expect(fields).To(ContainElement(&ast.Field{
				Names: []*ast.Ident{
					ast.NewIdent("methodMutex"),
				},
				Type: &ast.SelectorExpr{
					X:   ast.NewIdent("sync"),
					Sel: ast.NewIdent("RWMutex"),
				},
			}))
		})

		It("adds an invocations member to the struct", func() {
			// invocations map[string][][]interface{}
			Expect(genDecl.Specs).To(HaveLen(1))
			typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
			Expect(ok).To(BeTrue())

			structType, ok := typeSpec.Type.(*ast.StructType)
			Expect(ok).To(BeTrue())

			Expect(structType.Fields).NotTo(BeNil())
			Expect(structType.Fields.List).To(ContainElement(&ast.Field{
				Names: []*ast.Ident{ast.NewIdent("invocations")},
				Type: &ast.MapType{
					Key:   ast.NewIdent("string"),
					Value: ast.NewIdent("[][]interface{}"),
				},
			}))
		})

		It("adds an invocationsMutex member to the struct", func() {
			// invocationsMutex sync.RWMutex
			Expect(genDecl.Specs).To(HaveLen(1))
			typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
			Expect(ok).To(BeTrue())

			structType, ok := typeSpec.Type.(*ast.StructType)
			Expect(ok).To(BeTrue())

			Expect(structType.Fields).NotTo(BeNil())
			Expect(structType.Fields.List).To(ContainElement(&ast.Field{
				Names: []*ast.Ident{ast.NewIdent("invocationsMutex")},
				Type: &ast.SelectorExpr{
					X:   ast.NewIdent("sync"),
					Sel: ast.NewIdent("RWMutex"),
				},
			}))
		})

		It("adds a recordInvocation method to the funcDecls", func() {
			//  1: func (fake *FakeMyInterface) recordInvocation(key string, args []interface{}) {
			//  2:   fake.invocationsMutex.Lock()
			//  3:   defer fake.invocationsMutex.Unlock()
			//  4:   if fake.invocations == nil {
			//  5:   	fake.invocations = map[string][][]interface{}{}
			//  6:   }
			//  7:   if fake.invocations[key] == nil {
			//  8:   	fake.invocations[key] = [][]interface{}{}
			//  9:   }
			// 10:   fake.invocations[key] = append(fake.invocations[key], args)
			// 11: }
			var funcDecl *ast.FuncDecl
			for _, fn := range funcDecls {
				if fn.Name.Name == "recordInvocation" {
					funcDecl = fn
					break
				}
			}

			Expect(funcDecl).NotTo(BeNil())

			// line 1
			params := funcDecl.Type.Params.List
			Expect(params).To(HaveLen(2))

			Expect(params[0].Names[0].Name).To(Equal("key"))
			Expect(params[0].Type).To(Equal(ast.NewIdent("string")))

			Expect(params[1].Names[0].Name).To(Equal("args"))
			Expect(params[1].Type).To(Equal(ast.NewIdent("[]interface{}")))

			recv := funcDecl.Recv.List
			Expect(recv).To(HaveLen(1))
			Expect(recv[0].Names[0].Name).To(Equal("fake"))
			Expect(recv[0].Type).To(Equal(&ast.StarExpr{X: ast.NewIdent("FakeMyStruct")}))
			// end line 1

			bodyList := funcDecl.Body.List
			Expect(bodyList).To(HaveLen(5))

			// line 2
			line2, ok := bodyList[0].(*ast.ExprStmt)
			Expect(ok).To(BeTrue())

			_, ok = line2.X.(*ast.CallExpr)
			Expect(ok).To(BeTrue())

			///TODO: to be continued...
		})

		XIt("adds an Invocations method to the funcDecls", func() {
			// implemented, just not tested
		})

		XIt("does not contain a returns for each method that does not have return values", func() {
			// implemented, just not tested
		})

		XIt("does not contain an argsForCall for each method that does not have params", func() {
			// implemented, just not tested
		})

		Context("when a function in the interface takes params with an ellipsis", func() {
			BeforeEach(func() {
				src := []byte(`
package mypackage

type MyInterface interface {
	Method(...string)
}
`)
				var err error
				genDecl, funcDecls, err = patrick.Pour(src, "MyInterface", "MyStruct")
				Expect(err).NotTo(HaveOccurred())
			})

			It("creates the correct ArgsForCall struct member", func() {
				// methodArgsForCall []struct {
				//   arg1 ...string
				// }
				Expect(genDecl.Specs).To(HaveLen(1))
				typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
				Expect(ok).To(BeTrue())

				structType, ok := typeSpec.Type.(*ast.StructType)
				Expect(ok).To(BeTrue())

				Expect(structType.Fields).NotTo(BeNil())

				// can't use ConsistOf on structType.Fields due to the *ast.Ident
				// having NamePos values that are unknown
				var field *ast.Field
				for _, f := range structType.Fields.List {
					for _, name := range f.Names {
						if name.Name == "methodArgsForCall" {
							field = f
							break
						}
					}
				}

				Expect(field).NotTo(BeNil())

				arrayType, ok := field.Type.(*ast.ArrayType)
				Expect(ok).To(BeTrue())

				arrayStructType, ok := arrayType.Elt.(*ast.StructType)
				Expect(ok).To(BeTrue())
				fields := arrayStructType.Fields.List
				Expect(fields).To(HaveLen(1))

				fieldArrayType, ok := fields[0].Type.(*ast.ArrayType)
				Expect(ok).To(BeTrue())

				ident, ok := fieldArrayType.Elt.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(ident.Name).To(Equal("string"))
			})
		})

		Context("when a function in the interface has params", func() {
			BeforeEach(func() {
				src := []byte(`
package mypackage

type MyInterface interface {
	Method(int, string)
}
`)

				var err error
				genDecl, funcDecls, err = patrick.Pour(src, "MyInterface", "MyStruct")
				Expect(err).NotTo(HaveOccurred())
			})

			It("adds an ArgsForCall struct member for each method on the struct", func() {
				// methodArgsForCall []struct {
				//   arg1 int
				//   arg2 string
				// }
				Expect(genDecl.Specs).To(HaveLen(1))
				typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
				Expect(ok).To(BeTrue())

				structType, ok := typeSpec.Type.(*ast.StructType)
				Expect(ok).To(BeTrue())

				Expect(structType.Fields).NotTo(BeNil())

				// can't use ConsistOf on structType.Fields due to the *ast.Ident
				// having NamePos values that are unknown
				var field *ast.Field
				for _, f := range structType.Fields.List {
					for _, name := range f.Names {
						if name.Name == "methodArgsForCall" {
							field = f
							break
						}
					}
				}

				Expect(field).NotTo(BeNil())

				arrayType, ok := field.Type.(*ast.ArrayType)
				Expect(ok).To(BeTrue())

				arrayStructType, ok := arrayType.Elt.(*ast.StructType)
				Expect(ok).To(BeTrue())

				fields := arrayStructType.Fields.List
				Expect(fields).To(HaveLen(2))

				ident, ok := fields[0].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(ident.Name).To(Equal("int"))

				Expect(fields[0].Names).NotTo(BeNil())
				Expect(fields[0].Names[0].Name).To(Equal("arg1"))

				ident, ok = fields[1].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(ident.Name).To(Equal("string"))

				Expect(fields[1].Names).NotTo(BeNil())
				Expect(fields[1].Names[0].Name).To(Equal("arg2"))
			})
		})

		Context("when a function in the interface has return values", func() {
			BeforeEach(func() {
				src := []byte(`
package mypackage

type MyInterface interface {
	Method() (int, error)
}
`)

				var err error
				genDecl, funcDecls, err = patrick.Pour(src, "MyInterface", "MyStruct")
				Expect(err).NotTo(HaveOccurred())
			})

			It("adds a returns member for each method with return values to the struct", func() {
				// methodReturns struct {
				//   result1 int
				//   result2 error
				// }
				Expect(genDecl.Specs).To(HaveLen(1))
				typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
				Expect(ok).To(BeTrue())

				structType, ok := typeSpec.Type.(*ast.StructType)
				Expect(ok).To(BeTrue())

				Expect(structType.Fields).NotTo(BeNil())

				// can't use ConsistOf on structType.Fields due to the *ast.Ident
				// having NamePos values that are unknown
				var field *ast.Field
				for _, f := range structType.Fields.List {
					for _, name := range f.Names {
						if name.Name == "methodReturns" {
							field = f
							break
						}
					}
				}

				fieldStructType, ok := field.Type.(*ast.StructType)
				Expect(ok).To(BeTrue())

				returnsFields := fieldStructType.Fields.List
				Expect(returnsFields).To(HaveLen(2))

				ident, ok := returnsFields[0].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(ident.Name).To(Equal("int"))

				Expect(returnsFields[0].Names).NotTo(BeNil())
				Expect(returnsFields[0].Names[0].Name).To(Equal("result1"))

				ident, ok = returnsFields[1].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(ident.Name).To(Equal("error"))

				Expect(returnsFields[1].Names).NotTo(BeNil())
				Expect(returnsFields[1].Names[0].Name).To(Equal("result2"))
			})
		})
	})

	XIt("supports faking a struct with other fields present", func() {
		// to assert that we do not replace/modify existing fields
	})
})
