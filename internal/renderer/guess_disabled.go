//go:build !cgo || !onnxruntime || noguesser

package renderer

func Guess(_ string) string {
	// guesser disabled, should not be called
	panic("not implemented")
}
