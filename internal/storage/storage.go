package storage

import "io"

// Driver é o contrato que qualquer sistema de storage tem de cumprir.
// Seja disco local, S3 ou Google Drive.
type Driver interface {
	Put(key string, r io.Reader) error
	Get(key string) (io.ReadCloser, error)
	List() ([]string, error)
}
