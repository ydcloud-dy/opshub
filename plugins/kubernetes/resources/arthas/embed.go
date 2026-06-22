package arthas

import _ "embed"

//go:embed arthas-bin.tar
var packageTar []byte

// PackageTar returns the bundled Arthas distribution tarball.
func PackageTar() []byte {
	return packageTar
}
