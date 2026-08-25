package router

import "sync"

type seg struct {
	Value string
}

type kv struct {
	Key   string
	Value any
}

type Context struct {
	// Keep independently-owned pooled contexts from sharing an assumed CPU
	// cache line under parallel request load. See OPTIMIZATION.md.
	_ contextCachePad

	handler     *routeEntry
	segments    []seg
	store       []kv
	allowedMask int
}

func (c *Context) Set(key string, value any) {
	for i := 0; i < len(c.store); i++ {
		if c.store[i].Key == key {
			c.store[i].Value = value
			return
		}
	}

	c.store = append(c.store, kv{Key: key, Value: value})
}

func (c *Context) Get(key string) any {
	for i := 0; i < len(c.store); i++ {
		if c.store[i].Key == key {
			return c.store[i].Value
		}
	}

	return nil
}

func (c *Context) Param(key string) string {
	if c.handler == nil {
		return ""
	}

	parts := c.handler.Parts
	n := len(parts)
	if len(c.segments) < n {
		n = len(c.segments)
	}

	for i := 0; i < n; i++ {
		if parts[i] == key {
			return c.segments[i].Value
		}
	}

	return ""
}

func (c *Context) reset() {
	c.handler = nil
	c.allowedMask = 0

	if cap(c.segments) > 1024 {
		c.segments = make([]seg, 0, 8)
	} else {
		c.segments = c.segments[:0]
	}

	if cap(c.store) > 128 {
		c.store = make([]kv, 0, 4)
	} else {
		for i := range c.store {
			c.store[i] = kv{}
		}
		c.store = c.store[:0]
	}
}

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			segments: make([]seg, 0, 8),
			store:    make([]kv, 0, 4),
		}
	},
}

func getContext() *Context {
	ctx := contextPool.Get().(*Context)
	ctx.reset()
	return ctx
}

func putContext(ctx *Context) {
	contextPool.Put(ctx)
}
