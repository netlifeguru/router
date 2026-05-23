package router

import "testing"

func TestContextSetAndGet(t *testing.T) {
	ctx := getContext()
	defer putContext(ctx)

	if got := ctx.Get("missing"); got != nil {
		t.Fatalf("expected nil for missing key, got %#v", got)
	}

	ctx.Set("foo", "bar")
	if got := ctx.Get("foo"); got != "bar" {
		t.Fatalf("Get(%q) = %#v, want %q", "foo", got, "bar")
	}

	if len(ctx.store) != 1 {
		t.Fatalf("len(Store) = %d, want %d", len(ctx.store), 1)
	}

	ctx.Set("foo", "baz")
	if len(ctx.store) != 1 {
		t.Fatalf("len(Store) after overwrite = %d, want %d", len(ctx.store), 1)
	}

	if got := ctx.Get("foo"); got != "baz" {
		t.Fatalf("Get(%q) after overwrite = %#v, want %q", "foo", got, "baz")
	}

	ctx.Set("bar", 123)
	if got := ctx.Get("bar"); got != 123 {
		t.Fatalf("Get(%q) = %#v, want %v", "bar", got, 123)
	}
}

func TestContextParam(t *testing.T) {
	ctx := getContext()
	defer putContext(ctx)

	ctx.params = append(ctx.params,
		par{Key: "id", Value: "42"},
		par{Key: "slug", Value: "test-slug"},
	)

	if got := ctx.Param("id"); got != "42" {
		t.Fatalf("Param(%q) = %q, want %q", "id", got, "42")
	}

	if got := ctx.Param("slug"); got != "test-slug" {
		t.Fatalf("Param(%q) = %q, want %q", "slug", got, "test-slug")
	}

	if got := ctx.Param("missing"); got != "" {
		t.Fatalf("Param(%q) = %q, want empty string", "missing", got)
	}
}

func TestContextResetClearsFields(t *testing.T) {
	ctx := getContext()
	defer putContext(ctx)

	ctx.handler.Route = "/test"
	ctx.handler.Bitmask = 7
	ctx.handler.Validation = true
	ctx.handler.Parts = activeParts{"a", "b"}
	ctx.handler.Patterns = []pattern{{Slug: "s"}}
	ctx.fromCache = true

	ctx.params = append(ctx.params, par{Key: "id", Value: "1"})
	ctx.segments = append(ctx.segments, seg{Value: "seg"})
	ctx.entries = append(ctx.entries, routeEntry{Route: "/x"})
	ctx.store = append(ctx.store, kv{Key: "k", Value: "v"})

	ctx.reset()

	if ctx.handler.Route != "" {
		t.Fatalf("Handler.Route = %q, want empty", ctx.handler.Route)
	}
	if ctx.handler.Handler != nil {
		t.Fatalf("Handler.Handler not nil after reset")
	}
	if ctx.handler.Bitmask != 0 {
		t.Fatalf("Handler.Bitmask = %d, want 0", ctx.handler.Bitmask)
	}
	if ctx.handler.Validation {
		t.Fatalf("Handler.Validation = true, want false")
	}
	if len(ctx.handler.Parts) != 0 {
		t.Fatalf("len(Handler.Parts) = %d, want 0", len(ctx.handler.Parts))
	}
	if len(ctx.handler.Patterns) != 0 {
		t.Fatalf("len(Handler.Patterns) = %d, want 0", len(ctx.handler.Patterns))
	}

	if ctx.fromCache {
		t.Fatalf("fromCache = true, want false")
	}
	if len(ctx.params) != 0 {
		t.Fatalf("len(Params) = %d, want 0", len(ctx.params))
	}
	if len(ctx.segments) != 0 {
		t.Fatalf("len(Segments) = %d, want 0", len(ctx.segments))
	}
	if len(ctx.entries) != 0 {
		t.Fatalf("len(Entries) = %d, want 0", len(ctx.entries))
	}
	if len(ctx.store) != 0 {
		t.Fatalf("len(Store) = %d, want 0", len(ctx.store))
	}
}

func TestContextResetResizesLargeSlices(t *testing.T) {
	ctx := &Context{}

	ctx.params = make([]par, 0, 2048)
	ctx.segments = make([]seg, 0, 2048)
	ctx.entries = make([]routeEntry, 0, 2048)
	ctx.store = make([]kv, 0, 256)

	ctx.reset()

	if cap(ctx.params) != 8 {
		t.Fatalf("cap(Params) = %d, want 8", cap(ctx.params))
	}
	if cap(ctx.segments) != 8 {
		t.Fatalf("cap(Segments) = %d, want 8", cap(ctx.segments))
	}
	if cap(ctx.entries) != 8 {
		t.Fatalf("cap(Entries) = %d, want 8", cap(ctx.entries))
	}
	if cap(ctx.store) != 4 {
		t.Fatalf("cap(Store) = %d, want 4", cap(ctx.store))
	}
}
