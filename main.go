package main

import (
	"fmt"

	"golang.org/x/mod/semver"
)

func main() {
	fmt.Println(semver.Prerelease("v1.0.0-poop"))
}
