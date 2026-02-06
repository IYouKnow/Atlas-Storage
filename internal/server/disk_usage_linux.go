package server

import (
	"syscall"
)

// getDiskUsage returns the available and used bytes for the file system containing path.
func getDiskUsage(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	// Available blocks * block size
	free := stat.Bavail * uint64(stat.Bsize)
	// Total blocks * block size
	total := stat.Blocks * uint64(stat.Bsize)
	used := total - free
	return free, used, nil
}
