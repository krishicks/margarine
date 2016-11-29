package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"io/ioutil"
	"log"

	"golang.org/x/tools/imports"

	"github.com/krishicks/margarine"
)

func main() {
	opts := margarine.StructifyOpts{
		InterfaceName:      "Simple",
		RecvName:           "f",
		StructName:         "F",
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

	var buf bytes.Buffer
	err = format.Node(&buf, fset, f)
	if err != nil {
		panic(err)
	}

	out, err := imports.Process("src.go", buf.Bytes(), nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf(string(out))
}
