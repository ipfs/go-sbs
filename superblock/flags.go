package superblock

const (

	// insert flags here
	lastFlag = iota
)

const (
	reservedMask = (1<<16 - 1) & ^(1<<lastFlag - 1)
)
