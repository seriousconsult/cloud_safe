package utils

import (
	"sync"
)

// BufferPool provides a pool of reusable byte buffers for memory efficiency
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with the specified buffer size
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		size: size,
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if len(buf) == bp.size {
		bp.pool.Put(buf)
	}
}

// Size returns the buffer size
func (bp *BufferPool) Size() int {
	return bp.size
}
