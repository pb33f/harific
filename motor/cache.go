package motor

import "github.com/pb33f/harhar"

type NoOpCache struct{}

func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

func (c *NoOpCache) Get(index int) (*harhar.Entry, bool) {
	return nil, false
}

func (c *NoOpCache) Put(index int, entry *harhar.Entry) {}

func (c *NoOpCache) Clear() {}

func (c *NoOpCache) Size() int {
	return 0
}
