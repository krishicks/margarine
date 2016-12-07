package main

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func main() {
	// opts := patrick.Opts{
	// 	StructName:         "FakeSimple",
	// 	PreserveParamNames: false,
	// }

	// src, err := ioutil.ReadFile("notes.go")
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	// genDecl, funcDecls, err := patrick.Pour(src, "Simple", opts)
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	// decls := []ast.Decl{genDecl}
	// for _, fd := range funcDecls {
	// 	decls = append(decls, fd)
	// }

	// f := &ast.File{
	// 	Name: &ast.Ident{
	// 		Name: "mypackage",
	// 	},
	// 	Decls: decls,
	// }

	// fset := token.NewFileSet()
	// ast.Print(fset, f)

	// margarine.Fakify(f, margarine.FakifyOpts{StructName: "FakeSimple"})

	// -----------------

	src := `
	package somepackage

	type SomeStruct struct {
		ScanStub  func(int) bool
	}

	func (s SomeStruct) A() int {
		var a int
		return a
	}
	`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	ast.Print(fset, f)
	// -----------------
	// fset := token.NewFileSet()

	// var buf bytes.Buffer
	// err = format.Node(&buf, fset, f)
	// if err != nil {
	// 	panic(err)
	// }

	// out, err := imports.Process("src.go", buf.Bytes(), nil)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Printf(string(out))
}
