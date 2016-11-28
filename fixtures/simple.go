package fixtures

type Simple interface {
	A()
	// B(b int)
	// C(c int) (i int)

	// io.Writer
	// Embedded
}

type Embedded interface {
	EmbeddedA()
}
