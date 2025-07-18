package dataset

import (
	"os"
	"syscall"
)

func mmapSlice(f *os.File, off int64, size int64) ([]byte, error) {
	data, err := syscall.Mmap(int(f.Fd()), off, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return data, nil
}
