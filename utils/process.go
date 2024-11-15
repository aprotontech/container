package utils

import (
	"os"
	"syscall"
)

func IsProcessExists(pid int, cmd string) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err = process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	return true
}
