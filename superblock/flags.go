package superblock

const (

	// insert flags here
	lastFlag = 1 << iota
)

const (
	reservedMask = (1<<16 - 1) & ^(lastFlag - 1)
)
