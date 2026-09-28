//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// diskFree reports the free bytes available to the current user on the volume
// containing dir (used by the storage bar).
func diskFree(dir string) (int64, error) {
	var freeAvailable, total, totalFree uint64
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	if err := windows.GetDiskFreeSpaceEx(p, &freeAvailable, &total, &totalFree); err != nil {
		return 0, fmt.Errorf("app: disk free for %s: %w", dir, err)
	}
	return int64(freeAvailable), nil
}
