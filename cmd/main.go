package main

import (
	"fmt"
	"log"

	"github.com/krishicks/margarine"
)

func main() {
	src := `
	package somepackage

	import (
		"os"
	)

	type Simple interface {
		A(a, b int) int
		B() os.Signal
	}
	`

	opts := margarine.StructifyOpts{
		InterfaceName: "Simple",
		RecvName:      "f",
		StructName:    "F",
		PackageName:   "mypackage",
	}
	bs, err := margarine.Structify([]byte(src), opts)
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Printf(string(bs))
}
