package motor

import "github.com/pb33f/harific/motor/model"

type NoOpCache struct{}

func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

func (c *NoOpCache) Get(index int) (*model.Entry, bool) {
	return nil, false
}

func (c *NoOpCache) Put(index int, entry *model.Entry) {}

func (c *NoOpCache) Clear() {}

func (c *NoOpCache) Size() int {
	return 0
}
