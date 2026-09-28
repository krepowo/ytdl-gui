//go:build !windows

package main

import "fmt"

// diskFree is a non-Windows stub so the package still builds during
// cross-platform checks. The shipped app targets Windows only.
func diskFree(dir string) (int64, error) {
	return 0, fmt.Errorf("app: disk free is only implemented on Windows")
}
