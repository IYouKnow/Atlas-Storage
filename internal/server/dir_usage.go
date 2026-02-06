package server

import (
	"os"
	"path/filepath"
)

// getDirUsedBytes returns the total size in bytes of all files under dir (recursive).
// Used for quota reporting so we report space used by the share content, not the host disk.
func getDirUsedBytes(dir string) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += uint64(info.Size())
		return nil
	})
	return total, err
}
