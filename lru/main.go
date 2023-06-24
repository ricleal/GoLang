package main

import (
	"container/list"
	"fmt"
)

type lru struct {
	capacity int
	// list of keys
	queue *list.List
	data  map[string]*node
}

type node struct {
	Data   interface{}
	KeyPtr *list.Element
}

func (n *node) String() string {
	if key, ok := n.KeyPtr.Value.(string); ok {
		return fmt.Sprintf("%s - %v", key, n.Data)
	}
	return fmt.Sprintf("%v", n.Data)
}

func New(capacity int) *lru {
	return &lru{
		capacity: capacity,
		queue:    list.New(),
		data:     make(map[string]*node),
	}
}

func (c *lru) Add(key string, value interface{}) {
	if _, ok := c.data[key]; ok {
		fmt.Println("key already in the cache:", key)
		return
	}

	if len(c.data) >= c.capacity {
		// remove the oldest element
		elToRemove := c.queue.Front()
		c.queue.Remove(elToRemove)
		// check the type of the element
		if keyToRemove, ok := elToRemove.Value.(string); ok {
			delete(c.data, keyToRemove)
		} else {
			panic("invalid type: this should not happen")
		}
	}

	el := c.queue.PushBack(key)
	c.data[key] = &node{
		Data:   value,
		KeyPtr: el,
	}
}

func (c *lru) Get(key string) interface{} {
	// check if the key is in the map
	if val, ok := c.data[key]; ok {
		// move the key to the back of the queue
		el := val.KeyPtr
		c.queue.MoveToBack(el)
		return val.Data
	} else {
		fmt.Println("key not in the cache:", key)
		return nil
	}
}

func (c *lru) Print() {
	fmt.Println("--------------------")
	for el := c.queue.Front(); el != nil; el = el.Next() {
		fmt.Println(el.Value, ":", c.data[el.Value.(string)].Data)
	}
}

func main() {
	cache := New(3)
	cache.Add("1", 1)
	cache.Add("2", 2)
	cache.Add("3", 3)
	cache.Print()
	cache.Add("4", 4)
	// 1 should be removed
	cache.Print()
	v := cache.Get("2")
	fmt.Println("Got from cache:", v)
	cache.Print()
	cache.Add("5", 5)
	// 3 should be removed
	cache.Print()
}
