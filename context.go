package router

import (
	"sync"
)

type par struct {
	Key   string
	Value string
}

type seg struct {
	Value string
}

type kv struct {
	Key   string
	Value any
}

type Context struct {
	handler   routeEntry
	entries   []routeEntry
	params    []par
	segments  []seg
	store     []kv
	fromCache bool
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
	n := len(c.params)

	for i := 0; i < n; i++ {
		if c.params[i].Key == key {
			return c.params[i].Value
		}
	}

	return ""
}

func (c *Context) reset() {
	c.handler = routeEntry{}
	c.fromCache = false

	if cap(c.params) > 1024 {
		c.params = make([]par, 0, 8)
	} else {
		c.params = c.params[:0]
	}

	if cap(c.segments) > 1024 {
		c.segments = make([]seg, 0, 8)
	} else {
		c.segments = c.segments[:0]
	}

	if cap(c.entries) > 1024 {
		c.entries = make([]routeEntry, 0, 8)
	} else {
		c.entries = c.entries[:0]
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
			params:   make([]par, 0, 8),
			segments: make([]seg, 0, 8),
			store:    make([]kv, 0, 4),
			entries:  make([]routeEntry, 0, 8),
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
