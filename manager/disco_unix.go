//go:build !windows

package main

import "syscall"

// espacoEmDisco devolve (livre para o usuário, total) em bytes.
func espacoEmDisco(dir string) (int64, int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, 0, err
	}
	return int64(uint64(st.Bavail) * uint64(st.Bsize)), int64(uint64(st.Blocks) * uint64(st.Bsize)), nil
}
