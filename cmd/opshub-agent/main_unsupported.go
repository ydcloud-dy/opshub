//go:build !linux

package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("opshub-agent currently supports linux only, current platform is %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
