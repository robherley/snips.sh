package msgs

import "image/color"

// ThemeChanged is emitted when the user's chosen theme color is updated. Views
// that render with the theme color should re-style on receipt.
type ThemeChanged struct {
	Color color.Color
}
