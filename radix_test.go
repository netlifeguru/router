package router

import (
	"reflect"
	"strings"
	"testing"
)

func newTestRouterRadix() *Router {
	r := &Router{}
	r.radixRoot = &radixNode{}
	return r
}

func TestLongestCommonPrefixStr(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"", "abc", 0},
		{"abc", "abc", 3},
		{"abcd", "abxyz", 2},
		{"/users/*", "/users/123", len("/users/")},
	}

	for _, tt := range tests {
		got := longestCommonPrefixStr(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("longestCommonPrefixStr(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMatchNodePrefix_NoStarExactMatch(t *testing.T) {
	ctx := &Context{}
	prefix := "/users/"
	key := "/users/123"

	node := newRadixNode(prefix)
	consumed, ok := matchNodePrefix(node, key, ctx)
	if !ok {
		t.Fatalf("expected matchNodePrefix to match, got ok=false")
	}
	if consumed != len(prefix) {
		t.Errorf("expected consumed=%d, got %d", len(prefix), consumed)
	}
	if len(ctx.segments) != 0 {
		t.Errorf("expected no segments captured, got %d", len(ctx.segments))
	}
}

func TestMatchNodePrefix_StarCapturesSegment(t *testing.T) {
	ctx := &Context{}
	prefix := "/files/*/edit"
	key := "/files/foo/edit"

	node := newRadixNode(prefix)
	consumed, ok := matchNodePrefix(node, key, ctx)
	if !ok {
		t.Fatalf("expected matchNodePrefix to match, got ok=false")
	}

	if consumed != len(key) {
		t.Errorf("expected consumed=%d, got %d", len(key), consumed)
	}

	if len(ctx.segments) != 1 {
		t.Fatalf("expected 1 segment captured, got %d", len(ctx.segments))
	}
	if ctx.segments[0].Value != "foo" {
		t.Errorf("expected captured segment 'foo', got %q", ctx.segments[0].Value)
	}
}

func TestMatchNodePrefix_StarAtEndLeavesRemainder(t *testing.T) {
	ctx := &Context{}
	prefix := "/files/*"
	key := "/files/foo/bar"

	node := newRadixNode(prefix)
	consumed, ok := matchNodePrefix(node, key, ctx)
	if !ok {
		t.Fatalf("expected matchNodePrefix to match, got ok=false")
	}

	expectedConsumed := len("/files/foo")
	if consumed != expectedConsumed {
		t.Errorf("expected consumed=%d, got %d", expectedConsumed, consumed)
	}

	if len(ctx.segments) != 1 || ctx.segments[0].Value != "foo" {
		t.Errorf("expected captured segment 'foo', got %+v", ctx.segments)
	}
}

func TestMatchNodePrefix_NoMatch(t *testing.T) {
	ctx := &Context{}
	prefix := "/users/*"
	key := "/posts/123"

	node := newRadixNode(prefix)
	consumed, ok := matchNodePrefix(node, key, ctx)
	if ok {
		t.Fatalf("expected no match, got ok=true, consumed=%d", consumed)
	}
}

func TestRadixNodeAddChild_StarGlobAndNonStar(t *testing.T) {
	n := &radixNode{}

	starChild := newRadixNode("*wild")
	n.addChild(starChild)

	if len(n.star) != 1 {
		t.Fatalf("expected 1 star child, got %d", len(n.star))
	}
	if n.star[0] != starChild {
		t.Errorf("unexpected star child stored")
	}

	globChild := newRadixNode("**wild")
	n.addChild(globChild)

	if len(n.glob) != 1 {
		t.Fatalf("expected 1 glob child, got %d", len(n.glob))
	}
	if n.glob[0] != globChild {
		t.Errorf("unexpected glob child stored")
	}

	normalChild := newRadixNode("a")
	n.addChild(normalChild)

	idx := normalChild.prefix[0]
	if n.byFirst[idx] != normalChild {
		t.Errorf("unexpected child at byFirst[%d]", idx)
	}

	if len(n.usedIndices) != 1 || n.usedIndices[0] != idx {
		t.Errorf("expected usedIndices = [%d], got %v", idx, n.usedIndices)
	}
}

func TestRadixNodeRebuildIndex(t *testing.T) {
	parent := &radixNode{}

	c1 := newRadixNode("*wild")
	c2 := newRadixNode("a")
	c3 := newRadixNode("b")
	c4 := newRadixNode("**wild")

	children := []*radixNode{c1, c2, c3, c4}
	parent.rebuildIndex(children)

	if len(parent.star) != 1 || parent.star[0] != c1 {
		t.Errorf("expected star children [c1], got %+v", parent.star)
	}

	if len(parent.glob) != 1 || parent.glob[0] != c4 {
		t.Errorf("expected glob children [c4], got %+v", parent.glob)
	}

	if len(parent.usedIndices) != 2 {
		t.Fatalf("expected 2 usedIndices, got %d", len(parent.usedIndices))
	}

	if parent.byFirst['a'] != c2 {
		t.Errorf("expected child c2 at 'a', got %+v", parent.byFirst['a'])
	}
	if parent.byFirst['b'] != c3 {
		t.Errorf("expected child c3 at 'b', got %+v", parent.byFirst['b'])
	}
}

func TestRouterInsertAndSearch_SimpleRoute(t *testing.T) {
	r := newTestRouterRadix()

	entry := routeEntry{
		Bitmask: 1,
		Parts:   nil,
	}
	r.insertNode("/hello", entry)

	ctx := &Context{}

	res := r.search("/hello", ctx, 1)
	if res != 1 {
		t.Fatalf("expected search result 1, got %d", res)
	}

	if ctx.handler == nil || ctx.handler.Bitmask != 1 {
		t.Fatalf("expected handler bitmask 1, got %#v", ctx.handler)
	}
	if len(ctx.segments) != 0 {
		t.Errorf("expected no captured segments, got %v", ctx.segments)
	}
}

func TestRouterInsertAndSearch_WildcardRoute_Matched(t *testing.T) {
	r := newTestRouterRadix()

	entry := routeEntry{
		Bitmask: 1,
		Parts:   []string{"id"},
	}
	r.insertNode("/users/*", entry)

	ctx := &Context{}
	res := r.search("/users/123", ctx, 1)
	if res != 1 {
		t.Fatalf("expected search result 1, got %d", res)
	}

	if ctx.handler == nil || ctx.handler.Bitmask != 1 {
		t.Fatalf("expected handler bitmask 1, got %#v", ctx.handler)
	}
	if got := ctx.Param("id"); got != "123" {
		t.Errorf("expected Param(\"id\")=123, got %q", got)
	}
}

func TestRouterInsertAndSearch_GlobRoute_Matched(t *testing.T) {
	r := newTestRouterRadix()

	entry := routeEntry{
		Bitmask: 1,
		Parts:   []string{"wildcard"},
	}
	r.insertNode("/files/**", entry)

	ctx := &Context{}
	res := r.search("/files/images/logo.png", ctx, 1)
	if res != 1 {
		t.Fatalf("expected search result 1, got %d", res)
	}

	if ctx.handler == nil || ctx.handler.Bitmask != 1 {
		t.Fatalf("expected handler bitmask 1, got %#v", ctx.handler)
	}
	if got := ctx.Param("wildcard"); got != "images/logo.png" {
		t.Errorf("expected Param(\"wildcard\")=images/logo.png, got %q", got)
	}
}

func TestRouterInsertAndSearch_WildcardRoute_BitmaskMismatch(t *testing.T) {
	r := newTestRouterRadix()

	entry := routeEntry{
		Bitmask: 1,
		Parts:   []string{"id"},
	}
	r.insertNode("/users/*", entry)

	ctx := &Context{}
	res := r.search("/users/123", ctx, 2)
	if res != 2 {
		t.Fatalf("expected search result 2, got %d", res)
	}

	if ctx.allowedMask != 1 {
		t.Errorf("expected combined mask 1 in ctx.allowedMask, got %d", ctx.allowedMask)
	}
	if ctx.handler != nil {
		t.Errorf("expected no handler selected on bitmask mismatch, got %#v", ctx.handler)
	}
	if got := ctx.Param("id"); got != "" {
		t.Errorf("expected Param(\"id\") to be empty on bitmask mismatch, got %q", got)
	}
}

func TestRouterInsert_SplitsNodeOnPartialPrefix(t *testing.T) {
	r := newTestRouterRadix()

	entry1 := routeEntry{Bitmask: 1}
	entry2 := routeEntry{Bitmask: 2}

	r.insertNode("/abcde", entry1)
	r.insertNode("/abxyz", entry2)

	ctx1 := &Context{}
	res1 := r.search("/abcde", ctx1, 1)
	if res1 != 1 {
		t.Fatalf("expected res=1 for /abcde, got %d", res1)
	}

	ctx2 := &Context{}
	res2 := r.search("/abxyz", ctx2, 2)
	if res2 != 1 {
		t.Fatalf("expected res=1 for /abxyz, got %d", res2)
	}

	child := r.radixRoot.byFirst['/']
	if child == nil {
		t.Fatalf("expected child at root for '/'")
	}
	if !strings.HasPrefix("/abcde", child.prefix) && !strings.HasPrefix("/abxyz", child.prefix) {
		t.Errorf("expected child prefix to be common prefix, got %q", child.prefix)
	}
}

func TestMatchNodePrefix(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		key       string
		wantLen   int
		wantMatch bool
		wantSegs  []seg
	}{
		{
			name:      "Empty prefix and key",
			prefix:    "",
			key:       "",
			wantLen:   0,
			wantMatch: true,
		},
		{
			name:      "Empty prefix, non-empty key",
			prefix:    "",
			key:       "api/v1",
			wantLen:   0,
			wantMatch: false,
		},
		{
			name:      "Exact match without asterisk",
			prefix:    "api/v1/users",
			key:       "api/v1/users",
			wantLen:   12,
			wantMatch: true,
		},
		{
			name:      "Match prefix without asterisk",
			prefix:    "api/",
			key:       "api/v1/users",
			wantLen:   4,
			wantMatch: true,
		},
		{
			name:      "No match without asterisk",
			prefix:    "api/v2",
			key:       "api/v1",
			wantLen:   0,
			wantMatch: false,
		},
		{
			name:      "One star with suffix",
			prefix:    "users/*/posts",
			key:       "users/123/posts",
			wantLen:   15,
			wantMatch: true,
			wantSegs:  []seg{{Value: "123"}},
		},
		{
			name:      "One star at end",
			prefix:    "users/*",
			key:       "users/123",
			wantLen:   9,
			wantMatch: true,
			wantSegs:  []seg{{Value: "123"}},
		},
		{
			name:      "One star leaves remainder",
			prefix:    "users/*",
			key:       "users/123/posts",
			wantLen:   9,
			wantMatch: true,
			wantSegs:  []seg{{Value: "123"}},
		},
		{
			name:      "Glob at end",
			prefix:    "files/**",
			key:       "files/images/logo.png",
			wantLen:   21,
			wantMatch: true,
			wantSegs:  []seg{{Value: "images/logo.png"}},
		},
		{
			name:      "Glob not at end",
			prefix:    "files/**/bad",
			key:       "files/images/logo.png",
			wantLen:   0,
			wantMatch: false,
		},
		{
			name:      "Multiple parameters",
			prefix:    "orgs/*/users/*",
			key:       "orgs/google/users/john",
			wantLen:   22,
			wantMatch: true,
			wantSegs:  []seg{{Value: "google"}, {Value: "john"}},
		},
		{
			name:      "Wrong suffix after star",
			prefix:    "users/*/posts",
			key:       "users/123/comments",
			wantLen:   0,
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			node := newRadixNode(tt.prefix)
			gotLen, gotMatch := matchNodePrefix(node, tt.key, ctx)

			if gotLen != tt.wantLen {
				t.Errorf("matchLen = %v, want %v", gotLen, tt.wantLen)
			}
			if gotMatch != tt.wantMatch {
				t.Errorf("match = %v, want %v", gotMatch, tt.wantMatch)
			}

			if tt.wantSegs != nil {
				if !reflect.DeepEqual(ctx.segments, tt.wantSegs) {
					t.Errorf("ctx.segments = %v, want %v", ctx.segments, tt.wantSegs)
				}
			} else if len(ctx.segments) > 0 {
				t.Errorf("ctx.segments = %v, want empty", ctx.segments)
			}
		})
	}
}
