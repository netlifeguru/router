package router

type radixNode struct {
	prefix      string
	starAtPlus1 int

	byFirst [256]*radixNode
	star    []*radixNode
	glob    []*radixNode

	usedIndices []byte

	isLeaf  bool
	entries []routeEntry
}

func (n *radixNode) setPrefix(prefix string) {
	n.prefix = prefix
	n.starAtPlus1 = 0

	for i := 0; i < len(prefix); i++ {
		if prefix[i] == '*' {
			n.starAtPlus1 = i + 1
			return
		}
	}
}

func newRadixNode(prefix string) *radixNode {
	n := &radixNode{}
	n.setPrefix(prefix)
	return n
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
	}

	if n.byFirst[idx] == nil {
		n.usedIndices = append(n.usedIndices, idx)
	}
	n.byFirst[idx] = ch
}

func (n *radixNode) rebuildIndex(children []*radixNode) {
	for i := 0; i < len(n.usedIndices); i++ {
		n.byFirst[n.usedIndices[i]] = nil
	}
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

func matchNodePrefix(n *radixNode, key string, ctx *Context) (int, bool) {
	prefix := n.prefix
	pLen := len(prefix)
	kLen := len(key)

	if pLen == 0 {
		return 0, key == ""
	}

	if n.starAtPlus1 == 0 {
		if kLen < pLen || key[:pLen] != prefix {
			return 0, false
		}
		return pLen, true
	}

	origSegLen := len(ctx.segments)
	star := n.starAtPlus1 - 1

	if star > 0 {
		if kLen < star || key[:star] != prefix[:star] {
			return 0, false
		}
	}

	i, j := star, star
	for i < pLen {
		c := prefix[i]

		if c == '*' {
			if i+1 < pLen && prefix[i+1] == '*' {
				if i+2 != pLen {
					ctx.segments = ctx.segments[:origSegLen]
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
			ctx.segments = ctx.segments[:origSegLen]
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
	if child := node.byFirst[b]; child != nil {
		if r.tryInsertIntoChild(child, key, entry) {
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

	child := newRadixNode(key)
	child.isLeaf = true
	child.entries = []routeEntry{entry}
	node.addChild(child)
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
		newChild := newRadixNode(child.prefix[lcp:])
		newChild.isLeaf = child.isLeaf
		newChild.entries = child.entries

		moved := make([]*radixNode, 0, len(child.usedIndices)+len(child.star)+len(child.glob))
		for i := 0; i < len(child.usedIndices); i++ {
			idx := child.usedIndices[i]
			if ch := child.byFirst[idx]; ch != nil {
				moved = append(moved, ch)
			}
		}
		moved = append(moved, child.star...)
		moved = append(moved, child.glob...)

		newChild.rebuildIndex(moved)

		child.setPrefix(child.prefix[:lcp])
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

func (r *Router) dfs(n *radixNode, key string, ctx *Context, bitmask int) int {
	if len(key) == 0 {
		if n.isLeaf {
			var combinedMask int

			for ei := 0; ei < len(n.entries); ei++ {
				entry := &n.entries[ei]
				combinedMask |= entry.Bitmask

				if entry.Bitmask&bitmask != 0 {
					ctx.handler = entry
					return 1
				}
			}

			if len(n.entries) > 0 {
				ctx.allowedMask = combinedMask
				return 2
			}
		}
		return 0
	}

	if ch := n.byFirst[key[0]]; ch != nil {
		segLen := len(ctx.segments)
		if cons, ok := matchNodePrefix(ch, key, ctx); ok {
			if res := r.dfs(ch, key[cons:], ctx, bitmask); res > 0 {
				return res
			}
		}
		ctx.segments = ctx.segments[:segLen]
	}

	for i := 0; i < len(n.star); i++ {
		ch := n.star[i]
		segLen := len(ctx.segments)
		if cons, ok := matchNodePrefix(ch, key, ctx); ok {
			if res := r.dfs(ch, key[cons:], ctx, bitmask); res > 0 {
				return res
			}
		}
		ctx.segments = ctx.segments[:segLen]
	}

	for i := 0; i < len(n.glob); i++ {
		ch := n.glob[i]
		segLen := len(ctx.segments)
		if cons, ok := matchNodePrefix(ch, key, ctx); ok {
			if res := r.dfs(ch, key[cons:], ctx, bitmask); res > 0 {
				return res
			}
		}
		ctx.segments = ctx.segments[:segLen]
	}

	return 0
}
