package margarine

import "os"

type Simple interface {
	A(a, b int, c string, d int) (e, y int)
	B(int, int, int) os.Signal
	C(int, string, int)
}
