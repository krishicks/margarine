package main

import (
	"go/ast"
	"go/token"
	"io/ioutil"
	"log"

	"github.com/krishicks/margarine"
)

func main() {
	opts := margarine.StructifyOpts{
		InterfaceName:      "Simple",
		RecvName:           "f",
		StructName:         "FakeSimple",
		PackageName:        "mypackage",
		PreserveParamNames: true,
	}
	src, err := ioutil.ReadFile("notes.go")
	if err != nil {
		log.Fatal(err.Error())
	}

	f, err := margarine.Structify(src, opts)
	if err != nil {
		log.Fatal(err.Error())
	}

	fset := token.NewFileSet()
	ast.Print(fset, f)

	margarine.Fakify(f, margarine.FakifyOpts{StructName: "FakeSimple"})

	// -----------------

	// src := `
	// package somepackage

	// type SomeStruct struct {
	// ScanStub  func(int) bool
	// }
	// `

	// fset := token.NewFileSet()
	// f, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	// if err != nil {
	// 	panic(err)
	// }

	// ast.Print(fset, f)
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
