//go:build !linux

package server

// getDiskUsage returns dummy available and used bytes for non-Linux systems.
// This is to ensure the code compiles on Windows/Mac, although accurate reporting
// is only required for the Linux target as per instructions.
func getDiskUsage(path string) (uint64, uint64, error) {
	// Return arbitrary large values for development/testing on Windows
	return 100 * 1024 * 1024 * 1024, 0, nil // 100GB free, 0 used
}
