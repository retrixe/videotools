package main

import (
	"os"
	"slices"
)

const version = ""

var utility string = ""
var mainFn func() = func() {
	panic("Invalid compile-time configuration: no utility was selected for compilation")
}

func main() {
	// TODO: Bundle ffprobe/ffmpeg binaries?
	if slices.Contains(os.Args, "--version") || slices.Contains(os.Args, "-v") {
		println("videotools version " + version)
		println("Compiled for: " + utility)
		return
	}
	mainFn()
}
