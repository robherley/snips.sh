//go:build noguesser

package renderer

func Guess(_ string) string {
	// guesser disabled via build tag
	panic("not implemented")
}
