package main

import (
	"fmt"
	"log"
	"time"
)

var myCache *Cache

func main() {
	fmt.Println("Hello from LRU")

	// Illustrate the base case
	testBaseCase()

	testConcurrentCase()
}

func testConcurrentCase() {
	log.Println("Testing the concurrent case...")
	log.Println("========================")

	// Create the new cache, allow 15 readers and only 3 writers
	myCache = New(time.Duration(time.Minute*5), 10, 3, 2)

	// Add items overflowing to ensure we hit capacity
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("image-%d.jpg", i)
		// Items can be added to the set.
		myCache.Set(key, []byte("Some fake binary data"))
	}

	// Create 15 readers, only 3 should make progress at a time.
	// for i := 0; i < 15; i++ {
	// 	go func() {
	// 		data, ok := myCache.ConcurrentGet("image-1.jpg")
	// 		if ok {
	// 			fmt.Println("Found data: ", string(data))
	// 		}
	// 	}()
	// }

	// Create 5 writers, only 1 should can make progress at a time.
	for i := 0; i < 5; i++ {
		go func() {
			myCache.ConcurrentSet("image-99.jpg", []byte("some fake data"))
		}()
	}

	time.Sleep(time.Second * 60)
}

func testBaseCase() {
	log.Println("Testing the base case...")
	log.Println("========================")
	// Create the new cache
	myCache = New(time.Duration(time.Second*5), 10, 0, 0)

	// Add items overflowing to ensure we hit capacity
	for i := 0; i < 11; i++ {
		key := fmt.Sprintf("image-%d.jpg", i)
		// Items can be added to the set.
		err := myCache.Set(key, []byte("Some fake binary data"))
		if err != nil {
			log.Println("Oh oh!", err.Error())
		} else {
			log.Println("Added...")
		}
	}

	// Get an item out of the cache
	if result, exists := myCache.Get("image-1.jpg"); exists {
		fmt.Println(string(result))
	}

	// Wait for cached items to get evicted.
	time.Sleep(time.Second * 6)
}
