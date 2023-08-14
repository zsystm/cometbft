package mempool

import (
	"container/list"

	cmtsync "github.com/cometbft/cometbft/libs/sync"
	"github.com/cometbft/cometbft/types"
)

// TxCache defines an interface for transaction caching.
// Currently, a TxCache does not allow direct reading or getting of transaction
// values. A TxCache is used primarily to push transactions and removing
// transactions. Pushing via Push returns a boolean telling the caller if the
// transaction already exists in the cache or not.
type TxCache interface {
	// Reset resets the cache to an empty state.
	Reset()

	// Push adds the given transaction key to the cache and returns true if it was
	// newly added. Otherwise, it returns false.
	Push(txKey types.TxKey) bool

	// Remove removes the given transaction from the cache.
	Remove(txKey types.TxKey)

	// Has reports whether tx is present in the cache. Checking for presence is
	// not treated as an access of the value.
	Has(txKey types.TxKey) bool
}

var _ TxCache = (*LRUTxCache)(nil)

// LRUTxCache maintains a thread-safe LRU cache of transaction hashes (keys).
type LRUTxCache struct {
	mtx      cmtsync.Mutex
	size     int
	cacheMap map[types.TxKey]*list.Element
	list     *list.List
}

func NewLRUTxCache(cacheSize int) *LRUTxCache {
	return &LRUTxCache{
		size:     cacheSize,
		cacheMap: make(map[types.TxKey]*list.Element, cacheSize),
		list:     list.New(),
	}
}

// GetList returns the underlying linked-list that backs the LRU cache. Note,
// this should be used for testing purposes only!
func (c *LRUTxCache) GetList() *list.List {
	return c.list
}

func (c *LRUTxCache) Reset() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.cacheMap = make(map[types.TxKey]*list.Element, c.size)
	c.list.Init()
}

func (c *LRUTxCache) Push(txKey types.TxKey) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	moved, ok := c.cacheMap[txKey]
	if ok {
		c.list.MoveToBack(moved)
		return false
	}

	if c.list.Len() >= c.size {
		front := c.list.Front()
		if front != nil {
			frontKey := front.Value.(types.TxKey)
			delete(c.cacheMap, frontKey)
			c.list.Remove(front)
		}
	}

	e := c.list.PushBack(txKey)
	c.cacheMap[txKey] = e

	return true
}

func (c *LRUTxCache) Remove(txKey types.TxKey) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	e := c.cacheMap[txKey]
	delete(c.cacheMap, txKey)

	if e != nil {
		c.list.Remove(e)
	}
}

func (c *LRUTxCache) Has(txKey types.TxKey) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, ok := c.cacheMap[txKey]
	return ok
}

// NopTxCache defines a no-op transaction cache.
type NopTxCache struct{}

var _ TxCache = (*NopTxCache)(nil)

func (NopTxCache) Reset()                {}
func (NopTxCache) Push(types.TxKey) bool { return true }
func (NopTxCache) Remove(types.TxKey)    {}
func (NopTxCache) Has(types.TxKey) bool  { return false }
