//go:build arm64 || noguesser

package renderer

func Guess(content string) string {
	// currently not supporting guessing in arm64 bc of libtensorflow requirements
	// https://github.com/robherley/snips.sh/issues/39
	panic("not implemented")
}
