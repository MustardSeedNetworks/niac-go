//go:build linux

package truststore

func WriteAnchorFile(root, dst string, pem []byte) (string, error) {
	return writeAnchorFile(root, dst, pem)
}
