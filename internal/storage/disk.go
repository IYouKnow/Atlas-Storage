package storage

import (
	"io"
	"os"
	"path/filepath"
)

type DiskDriver struct {
	RootPath string
}

// Garante que DiskDriver cumpre a interface Driver
var _ Driver = (*DiskDriver)(nil)

func NewDiskDriver(path string) *DiskDriver {
	os.MkdirAll(path, 0755) // Cria a pasta se não existir
	return &DiskDriver{RootPath: path}
}

func (d *DiskDriver) Put(key string, r io.Reader) error {
	fullPath := filepath.Join(d.RootPath, key)

	// Cria o ficheiro
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Grava os dados (Stream)
	_, err = io.Copy(f, r)
	return err
}

func (d *DiskDriver) Get(key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(d.RootPath, key)
	return os.Open(fullPath)
}

func (d *DiskDriver) List() ([]string, error) {
	entries, err := os.ReadDir(d.RootPath)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}
