//go:build !windows

package sim

func openSharedMemWriter() (*SharedMem, error) { return nil, nil }
func openSharedMemReader() (*SharedMem, error) { return nil, nil }
func closeSharedMem(s *SharedMem)              {}