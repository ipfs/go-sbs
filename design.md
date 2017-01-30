# fsbs, The Fast Static Blob Store

## Problem
Filesystems and existing databases are not optimized for storing relatively
small, static, content addressed blobs of data. IPFS has a dire need of a
storage system that excells at storing this class of data.

## Design Considerations
- Values being stored are static and will not be modified (write only)
- Keys are binary and can contain any byte (not unix paths)
- Should optimize for small files (< 256 bytes) and 'average' blobs (256k to
  4MB)
- Large value perf ( > 4MB ) should not be prioritized
- Keys are relatively uniform in size (though maybe this one might not be
  valid)
- Should work well directly on a disk or as a file on an existing filesystem
- Must support concurrent reads and writes
- defrag process should be incremental (or automatic)
- Try to use relative block indexing to not limit ourselves on maximum
  filesystem size
- Seeking through values is not a high priority feature. Can optimize for full
  value sequential reads

## Design Overview

The base layer of fsbs is the block allocator. The fsbs block allocator uses a
bitfield to allocate ranges of 8k blocks for the rest of the system. 
The next layer is the metadata shard. This object is a HAMT node and contains a
header and an array of fixed size key records as well as an adjoining block
containing an array for the actual keys being stored.
The metadata key records contain information about the value 'type', its
location on disk and its size.
Each of these parts is discussed in detail in its respective section.

The HAMT algorithm is used to find keys and build the tree. Blake2b 512 is the
selected hash function for HAMT traversal.

### FSBS Block Allocator
Each allocator is an 8k block containing a small header, and a bitfield to
track which blocks are allocated. The first block in the fsbs is the first
allocator and each following allocator is placed immediately after the range of
blocks the previous allocator is responsible for. This way each allocator is
predictably placed on disk and can be seeked to easily without extra
information needed.

#### Header
The header contains the following information:

| Field | Size | Description |
| ----- | ---- | ----------- |
| Version | 1 | Specifies the version of this allocator object |
| Blocks In Use | 3 | Denotes the number of blocks currently in use in this allocator |
| Last Allocator | 8 | Denotes the block index of the 'latest' allocator |
| Flag | 1 | Used to mark various traits of this allocator (eg. fragmentation) |
| Free Block Counter | 3 | number of entries in the free blocks list | 
| Free Blocks List | 16 or 48 | limited size array of freed block ranges |

The rest of the allocator block is a bitfield for tracking allocation.

Note: The 'Last Allocator' field could be omitted and we could simply scan for
the 'last allocator' and hold it in memory when initially opening up the
datastore.

#### Operations 
The allocator has two primary operations, `Allocate(n)` and `Free([x])`.

`Allocate` allocates and returns `n` blocks to the caller. In the simplest
case, the allocator has enough blocks and is not fragmented. In this case, the
next `n` contiguous blocks in the allocator are marked as in use, the `Blocks
In Use` field is updated, and the block range is returned to the caller.

TODO: case when fragmented.

When the current allocator is full or does not have enough blocks to satisfy
the call to `Allocate`, The process allocates all the blocks it can from the
current allocator, then skips to the next allocator (updating the `Last
Allocator` header field to aid in future operations). The remaining blocks are
then allocated from the next allocator block and returned.

`Free` is used to mark an array of blocks as no longer being used. Each freed
block is marked freed in the bitfield of its allocator, and the allocators
`Blocks In Use` field is updated. If the allocator a block is removed from is
full, then the allocator is marked as fragmented. If the allocator is not full,
it will be marked as fragmented once it is filled to avoid unecessary
performance degradation. If the `Free Blocks List` is not full, the freed
blocks are appended to the end of the list to make fragmented allocations
faster in the future. If the list is full then the freed blocks may be found
later by scanning the bitfield once the current free blocks list has been
consumed.

TODO: defragmentation. (note: should take care to make the process easily incremental)

### Metadata HAMT
Instead of using B-Trees for managing keys, fsbs uses a Hash Array Mapped Trie
to store key/value mappings. The Metadata block is a HAMT node and contains a
header, a small block of space for either a bloom filter or another optimizing
structure (NOTE: this is still being thought through), and an array of 512 Key
Records.

The header contains the following information:

| Field | Size | Description |
| ----- | ---- | ----------- |
| Version | 1 | Specifies the version of this metadata block |
| Key Size | ? | Specifies the size of keys being managed by this block |
| Key Array Blocks | ? | A list of blocks being used to store the key array for this block | 
| Block Index Base | 8 | The base index offset for blocks referenced here |
| First Listing Block | 8 | index of first listing block |
| Last Listing Block | 8 | index of latest listing block |

#### Key Records
The Metadata block contains an array of 512 Key Records. Each key record has the following format:

| Field | Size | Description |
| ----- | ---- | ----------- |
| Flag | 1 | Specifies the type of this record |
| Size | 7 | Unsigned integer, specifies the size of this record |
| BlockRel | 2 | Specifies the block index offset for this record | 
| Offset | 2 | Specifies the offset this record is located at within the block |

The flag may be one of the following values:

| Value | Meaning |
| ----- | ------- |
| 0 | Empty |
| 1 | Tiny Value |
| 2 | Direct Value |
| 3 | Block Range |
| 4 | Block Trie |
| 5 | Shard |

- Tiny Value
	- Values of this type are stored directly in the listing block
- Direct Value
	- This value consists of only one block, and that block is denoted in the key record 'BlockRel' field
- Block Range
	- The BlockRel value denotes the starting block of a range of blocks storing this value
- Block Trie
	- The listing block entry for this record contains a Block Trie of block indexes for this value
	- This is only used for large value storage
	- TODO: define the format for this
- Shard
	- This record points to another Metadata block (child node in the HAMT)

#### Listing Blocks
Listing blocks are accessory blocks to the metadata blocks. They are used to
store both tiny files and indirect block sets for values larger than a single
block.

There is a small header at the beginning of these blocks, that contains the following:

| Field | Size | Description |
| ----- | ---- | ----------- |
| Flag | 1 | Specifies flags for this block |
| UsedSize | 2 | Unsigned integer, denotes how much space in this block has been used |
| Head | 2 | Unsigned integer, denotes the offset in the block to be written to next |

The only value currently used in the flags field is the lowest bit. If set, it
means the listing block is fragmented.


Implementation Thoughts:
Right now, i'm thinking of having 512 way sharding. Each shard entry being 12
bytes means this array takes up 6144 bytes. We could potentially add more
metadata to each record and bring the size up to 16 bytes, then move the
sharding the 256 way. Doing that makes the array take up 4096 bytes and leaves
an uncomfortably large amount of free space in the metadata node. One though is
to use the free space (in either case) for a bloom filter, but that bloom
filter can contain around 1000 entries before it becomes not useful (more if
we're okay with a higher false positive rate). Once we're well out of the range
of having the bloom filters be useful, we could think of something more useful
to put in that space to help optimize certain actions.

Another thing to note is that the 'BlockRel' field is only two bytes, meaning
that we can only reference 2^16 blocks contiguously following the Block Index
Base specified in the header. Care must be taken to make sure we can do this
properly (and not have other parts of the system use up all the blocks in our
range). The alternative is to not use offsets here, but doing so means that we
have to have a large integer to represent block indexes, 8 bytes for a 64 bit
integer seems where we would start for this but then we're limited to 2^64
blocks we can allocate, and then the key records now have to be 18 bytes each.

When a value is stored in a key record, the key is stored in an adjoining
block. My initial idea is to have a fixed size for keys, which allows us to use
the HAMT array index to index into the key array to read the key. We don't want
to store the keys in the key records directly as doing so would make it quite
hard to keep the key records a fixed size (which is almost a requirement for
performant hamt traversal). The downside of a fixed key array is that there
will be sizeable wasted space once a nodes children are mostly shards. One
potential solution to this is to store records at each layer and keep
accounting for child shards separate. This means that lookups don't need to
traverse all the way to the leaf nodes in some cases to find their record.
