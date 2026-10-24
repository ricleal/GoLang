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
	if c.capacity <= 0 {
		return
	}

	if existing, ok := c.data[key]; ok {
		existing.Data = value
		c.queue.MoveToBack(existing.KeyPtr)
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
	}
	return nil
}

func (c *lru) Print() {
	fmt.Println("--------------------")
	for el := c.queue.Front(); el != nil; el = el.Next() {
		fmt.Println(el.Value, ":", c.data[el.Value.(string)].Data)
	}
}

// lruEnhanced uses a map-backed doubly linked list instead of container/list.
type enhancedNode struct {
	key      string
	value    interface{}
	next     *enhancedNode
	previous *enhancedNode
}

type lruEnhanced struct {
	capacity int
	data     map[string]*enhancedNode
	head     *enhancedNode
	tail     *enhancedNode
}

func NewEnhanced(capacity int) *lruEnhanced {
	return &lruEnhanced{
		capacity: capacity,
		data:     make(map[string]*enhancedNode),
	}
}

func (c *lruEnhanced) Add(key string, value interface{}) {
	if c.capacity <= 0 {
		return
	}

	if existing, ok := c.data[key]; ok {
		existing.value = value
		c.moveToBack(existing)
		return
	}

	if len(c.data) >= c.capacity {
		c.remove(c.head)
	}

	n := &enhancedNode{key: key, value: value}
	c.data[key] = n
	c.append(n)
}

func (c *lruEnhanced) Get(key string) interface{} {
	n, ok := c.data[key]
	if !ok {
		return nil
	}
	c.moveToBack(n)
	return n.value
}

func (c *lruEnhanced) Print() {
	fmt.Println("--------------------")
	for n := c.head; n != nil; n = n.next {
		fmt.Println(n.key, ":", n.value)
	}
}

func (c *lruEnhanced) append(n *enhancedNode) {
	n.previous = c.tail
	n.next = nil
	if c.tail == nil {
		c.head = n
	} else {
		c.tail.next = n
	}
	c.tail = n
}

func (c *lruEnhanced) remove(n *enhancedNode) {
	if n == nil {
		return
	}
	if n.previous == nil {
		c.head = n.next
	} else {
		n.previous.next = n.next
	}
	if n.next == nil {
		c.tail = n.previous
	} else {
		n.next.previous = n.previous
	}
	delete(c.data, n.key)
	n.next = nil
	n.previous = nil
}

func (c *lruEnhanced) moveToBack(n *enhancedNode) {
	if c.tail == n {
		return
	}
	if n.previous == nil {
		c.head = n.next
	} else {
		n.previous.next = n.next
	}
	n.next.previous = n.previous
	c.append(n)
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
