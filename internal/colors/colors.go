package colors

import "github.com/TwiN/go-color"

const darkGrey = "\033[38;2;90;90;90m"
const darkYellow = "\033[38;2;128;128;0m"

func InDarkGrey(text string) string {
	return color.Colorize(darkGrey, text)
}

func InDarkYellow(text string) string {
	return color.Colorize(darkYellow, text)
}
