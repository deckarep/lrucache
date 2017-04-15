package main // TODO: move to it's own package.

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type BinaryItem struct {
	sync.RWMutex

	data   []byte
	expiry *time.Time
}

func (item *BinaryItem) livelonger(duration time.Duration) {
	item.Lock()
	defer item.Unlock()

	exp := time.Now().Add(duration)
	item.expiry = &exp
}

func (item *BinaryItem) isExpired() bool {
	var value bool
	item.RLock()
	if item.expiry == nil {
		value = true
	} else {
		value = item.expiry.Before(time.Now())
	}
	item.RUnlock()
	return value
}

type Cache struct {
	sync.RWMutex
	ttl     time.Duration
	items   map[string]*BinaryItem
	maxSize int

	writeConcurrency int
	writeBounds      chan int

	readConcurrency int
	readBounds      chan int
}

func New(duration time.Duration, maxSize, readConcurrency, writeConcurrency int) *Cache {
	cache := &Cache{
		ttl:              duration,
		items:            make(map[string]*BinaryItem, 0),
		maxSize:          maxSize,
		readConcurrency:  readConcurrency,
		readBounds:       make(chan int, readConcurrency),
		writeConcurrency: writeConcurrency,
		writeBounds:      make(chan int, writeConcurrency),
	}
	cache.startEvictionPoll()
	return cache
}

// Every ticket it will evict old items.
func (cache *Cache) startEvictionPoll() {
	duration := cache.ttl

	ticker := time.Tick(duration)
	go func() {
		for {
			select {
			case <-ticker:
				cache.doEvict()
			}
		}
	}()
}

// Spins through the internal map of items and evicts.
func (cache *Cache) doEvict() {
	cache.Lock()
	defer cache.Unlock()

	for key, item := range cache.items {
		if item.isExpired() {
			log.Println("Evicting key because expired: ", key)
			delete(cache.items, key)
		}
	}
}

func (cache *Cache) ConcurrentGet(key string) ([]byte, bool) {
	cache.readBounds <- 1

	fmt.Println("Executing concurrent get...")

	var result []byte
	var found bool

	// This is a bit convulted because I wanted to show that it works.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		result, found = cache.Get(key)
		//Simulate network call perhaps or RPC
		time.Sleep(time.Second * 5)
		<-cache.readBounds
		wg.Done()
	}()

	wg.Wait()
	return result, found
}

func (cache *Cache) Get(key string) ([]byte, bool) {
	cache.Lock()
	defer cache.Unlock()

	item, ok := cache.items[key]
	if !ok || item.isExpired() {
		return nil, false
	}

	item.livelonger(cache.ttl)
	return item.data, true
}

func (cache *Cache) ConcurrentSet(key string, data []byte) error {
	cache.writeBounds <- 1

	fmt.Println("Executing concurrent set...")

	var err error

	// This is a bit convulted because I wanted to show that it works.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err = cache.Set(key, data)
		//Simulate network call perhaps or RPC
		time.Sleep(time.Second * 5)
		<-cache.writeBounds
		wg.Done()
	}()

	wg.Wait()
	return err
}

func (cache *Cache) Set(key string, data []byte) error {
	cache.Lock()
	defer cache.Unlock()

	if len(cache.items) == cache.maxSize {
		return fmt.Errorf("can't add more than %d items: exceeded capacity of cache structure.", cache.maxSize)
	}

	item := &BinaryItem{data: data}
	item.livelonger(cache.ttl)
	cache.items[key] = item

	return nil
}
