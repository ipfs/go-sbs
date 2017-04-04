package consts

const (
	AllocatorHeaderSize = 64
	BlocksPerAllocator  = (BlockSize - AllocatorHeaderSize) * 8
)
