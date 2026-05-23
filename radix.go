package router

type radixNode struct {
	prefix string

	byFirst [256][]*radixNode
	star    []*radixNode
	glob    []*radixNode

	usedIndices []byte

	isLeaf  bool
	entries []routeEntry
}

func (n *radixNode) addChild(ch *radixNode) {
	if len(ch.prefix) > 0 && ch.prefix[0] == '*' {
		if len(ch.prefix) >= 2 && ch.prefix[1] == '*' {
			n.glob = append(n.glob, ch)
		} else {
			n.star = append(n.star, ch)
		}
		return
	}

	var idx byte
	if len(ch.prefix) > 0 {
		idx = ch.prefix[0]
	} else {
		idx = 0
	}

	if len(n.byFirst[idx]) == 0 {
		n.usedIndices = append(n.usedIndices, idx)
	}

	n.byFirst[idx] = append(n.byFirst[idx], ch)
}

func (n *radixNode) rebuildIndex(children []*radixNode) {
	n.byFirst = [256][]*radixNode{}
	n.usedIndices = n.usedIndices[:0]
	n.star = n.star[:0]
	n.glob = n.glob[:0]

	for i := 0; i < len(children); i++ {
		n.addChild(children[i])
	}
}

func longestCommonPrefixStr(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func matchPrefixWithStarStr(prefix, key string, ctx *Context) (int, bool) {
	pLen := len(prefix)
	kLen := len(key)

	if pLen == 0 {
		return 0, key == ""
	}

	origSegLen := len(ctx.segments)
	i, j := 0, 0

	for i < pLen {
		c := prefix[i]

		if c == '*' {
			if i+1 < pLen && prefix[i+1] == '*' {
				if i+2 != pLen {
					ctx.segments = ctx.segments[:origSegLen] // Rollback
					return 0, false
				}

				ctx.segments = append(ctx.segments, seg{Value: key[j:]})
				return kLen, true
			}

			start := j
			for j < kLen && key[j] != '/' {
				j++
			}
			ctx.segments = append(ctx.segments, seg{Value: key[start:j]})
			i++
			continue
		}

		if j >= kLen || c != key[j] {
			ctx.segments = ctx.segments[:origSegLen] // Rollback
			return 0, false
		}
		i++
		j++
	}

	return j, true
}

func (r *Router) insertNode(key string, entry routeEntry) {
	r.insert(r.radixRoot, key, entry)
}

func (r *Router) insert(node *radixNode, key string, entry routeEntry) {
	if len(key) == 0 {
		node.isLeaf = true
		node.entries = append(node.entries, entry)
		return
	}

	b := key[0]

	c := node.byFirst[b]
	for i := 0; i < len(c); i++ {
		if r.tryInsertIntoChild(c[i], key, entry) {
			return
		}
	}

	for i := 0; i < len(node.star); i++ {
		if r.tryInsertIntoChild(node.star[i], key, entry) {
			return
		}
	}

	for i := 0; i < len(node.glob); i++ {
		if r.tryInsertIntoChild(node.glob[i], key, entry) {
			return
		}
	}

	node.addChild(&radixNode{
		prefix:  key,
		isLeaf:  true,
		entries: []routeEntry{entry},
	})
}

func (r *Router) tryInsertIntoChild(child *radixNode, key string, entry routeEntry) bool {
	if len(child.prefix) == 0 {
		return false
	}
	if child.prefix[0] != '*' && key[0] != child.prefix[0] {
		return false
	}

	lcp := longestCommonPrefixStr(child.prefix, key)
	if lcp == 0 {
		return false
	}

	if lcp == len(child.prefix) && lcp == len(key) {
		child.isLeaf = true
		child.entries = append(child.entries, entry)
		return true
	}

	if lcp < len(child.prefix) {
		newChild := &radixNode{
			prefix:  child.prefix[lcp:],
			isLeaf:  child.isLeaf,
			entries: child.entries,
		}

		var moved []*radixNode
		for i := 0; i < len(child.usedIndices); i++ {
			idx := child.usedIndices[i]
			moved = append(moved, child.byFirst[idx]...)
		}
		moved = append(moved, child.star...)
		moved = append(moved, child.glob...)

		newChild.rebuildIndex(moved)

		child.prefix = child.prefix[:lcp]
		child.isLeaf = false
		child.entries = nil
		child.rebuildIndex([]*radixNode{newChild})
	}

	if lcp < len(key) {
		r.insert(child, key[lcp:], entry)
	} else {
		child.isLeaf = true
		child.entries = append(child.entries, entry)
	}
	return true
}

func (r *Router) search(key string, ctx *Context, bitmask int) int {
	return r.dfs(r.radixRoot, key, ctx, bitmask)
}

func (r *Router) dfs(n *radixNode, k string, ctx *Context, bitmask int) int {
	if len(k) == 0 {
		if n.isLeaf {
			var combinedMask int

			for ei := 0; ei < len(n.entries); ei++ {
				combinedMask |= n.entries[ei].Bitmask

				if n.entries[ei].Bitmask&bitmask != 0 {
					ctx.handler = n.entries[ei]

					pn := len(ctx.handler.Parts)
					sn := len(ctx.segments)
					if sn < pn {
						pn = sn
					}

					for pi := 0; pi < pn; pi++ {
						ctx.params = append(ctx.params, par{
							Key:   ctx.handler.Parts[pi],
							Value: ctx.segments[pi].Value,
						})
					}
					return 1
				}
			}

			if len(n.entries) > 0 {
				ctx.handler.Bitmask = combinedMask
				return 2
			}
		}
		return 0
	}

	children := n.byFirst[k[0]]
	for i := 0; i < len(children); i++ {
		ch := children[i]

		segLen := len(ctx.segments)
		if cons, ok := matchPrefixWithStarStr(ch.prefix, k, ctx); ok {
			if res := r.dfs(ch, k[cons:], ctx, bitmask); res > 0 {
				return res
			}
		}
		ctx.segments = ctx.segments[:segLen]
	}

	for i := 0; i < len(n.star); i++ {
		ch := n.star[i]

		segLen := len(ctx.segments)
		if cons, ok := matchPrefixWithStarStr(ch.prefix, k, ctx); ok {
			if res := r.dfs(ch, k[cons:], ctx, bitmask); res > 0 {
				return res
			}
		}
		ctx.segments = ctx.segments[:segLen]
	}

	for i := 0; i < len(n.glob); i++ {
		ch := n.glob[i]

		segLen := len(ctx.segments)
		if cons, ok := matchPrefixWithStarStr(ch.prefix, k, ctx); ok {
			if res := r.dfs(ch, k[cons:], ctx, bitmask); res > 0 {
				return res
			}
		}
		ctx.segments = ctx.segments[:segLen]
	}

	return 0
}
