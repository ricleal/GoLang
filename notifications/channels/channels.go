package channels

import "sync"

type Channels struct {
	data     map[string]chan struct{}
	m        sync.Mutex
	nBuffers int
}

// Add adds a new channel to the map.
func (c *Channels) Add(sid string) {
	c.m.Lock()
	defer c.m.Unlock()
	c.data[sid] = nil
}

// Send sends a message to the channel.
func (c *Channels) Send(sid string) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.data[sid] == nil {
		c.data[sid] = make(chan struct{}, c.nBuffers)
	}
	c.data[sid] <- struct{}{}
}

func (c *Channels) Exists(sid string) bool {
	c.m.Lock()
	defer c.m.Unlock()
	if _, ok := c.data[sid]; !ok {
		return false
	}
	return true
}

// Keys returns a slice of keys from the map.
func (c *Channels) Keys() []string {
	c.m.Lock()
	defer c.m.Unlock()
	var keys []string
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Get returns a channel from the map.
func (c *Channels) Get(sid string) <-chan struct{} {
	c.m.Lock()
	defer c.m.Unlock()
	if _, ok := c.data[sid]; !ok {
		return nil
	}
	return c.data[sid]
}

// Remove removes a key from the map.
func (c *Channels) Remove(sid string) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.data, sid)
}

func (c *Channels) Close(sid string) {
	c.m.Lock()
	defer c.m.Unlock()
	close(c.data[sid])
}

func (c *Channels) Shutdown(sid string) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.data[sid] != nil {
		close(c.data[sid])
	}
	delete(c.data, sid)
}

func NewChannels(nBuffers int) *Channels {
	return &Channels{
		nBuffers: nBuffers,
		// make the map
		data: make(map[string](chan struct{})),
	}
}
