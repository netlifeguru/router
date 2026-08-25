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

	entry := routeEntry{Parts: activeParts{"id", "slug"}}
	ctx.handler = &entry
	ctx.segments = append(ctx.segments,
		seg{Value: "42"},
		seg{Value: "test-slug"},
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

func TestContextParamWithoutHandler(t *testing.T) {
	ctx := getContext()
	defer putContext(ctx)

	ctx.segments = append(ctx.segments, seg{Value: "42"})
	if got := ctx.Param("id"); got != "" {
		t.Fatalf("Param(%q) = %q without handler, want empty string", "id", got)
	}
}

func TestContextParamUsesShortestPartsAndSegmentsLength(t *testing.T) {
	ctx := getContext()
	defer putContext(ctx)

	entry := routeEntry{Parts: activeParts{"id", "slug"}}
	ctx.handler = &entry
	ctx.segments = append(ctx.segments, seg{Value: "42"})

	if got := ctx.Param("id"); got != "42" {
		t.Fatalf("Param(%q) = %q, want %q", "id", got, "42")
	}
	if got := ctx.Param("slug"); got != "" {
		t.Fatalf("Param(%q) = %q with missing segment, want empty string", "slug", got)
	}
}

func TestContextResetClearsFields(t *testing.T) {
	ctx := getContext()
	defer putContext(ctx)

	entry := routeEntry{
		Route:      "/test",
		Bitmask:    7,
		Validation: true,
		Parts:      activeParts{"a", "b"},
		Patterns:   []pattern{{Slug: "s"}},
	}
	ctx.handler = &entry
	ctx.allowedMask = 15
	ctx.segments = append(ctx.segments, seg{Value: "seg"})
	ctx.store = append(ctx.store, kv{Key: "k", Value: "v"})

	ctx.reset()

	if ctx.handler != nil {
		t.Fatalf("handler = %#v, want nil", ctx.handler)
	}
	if ctx.allowedMask != 0 {
		t.Fatalf("allowedMask = %d, want 0", ctx.allowedMask)
	}
	if len(ctx.segments) != 0 {
		t.Fatalf("len(Segments) = %d, want 0", len(ctx.segments))
	}
	if len(ctx.store) != 0 {
		t.Fatalf("len(Store) = %d, want 0", len(ctx.store))
	}
}

func TestContextResetResizesLargeSlices(t *testing.T) {
	ctx := &Context{}

	ctx.segments = make([]seg, 0, 2048)
	ctx.store = make([]kv, 0, 256)

	ctx.reset()

	if cap(ctx.segments) != 8 {
		t.Fatalf("cap(Segments) = %d, want 8", cap(ctx.segments))
	}
	if cap(ctx.store) != 4 {
		t.Fatalf("cap(Store) = %d, want 4", cap(ctx.store))
	}
}

func TestContextResetRetainsReasonableCapacity(t *testing.T) {
	ctx := &Context{
		segments: make([]seg, 0, 16),
		store:    make([]kv, 0, 8),
	}
	ctx.segments = append(ctx.segments, seg{Value: "x"})
	ctx.store = append(ctx.store, kv{Key: "k", Value: "v"})

	ctx.reset()

	if cap(ctx.segments) != 16 {
		t.Fatalf("cap(Segments) = %d, want retained capacity 16", cap(ctx.segments))
	}
	if cap(ctx.store) != 8 {
		t.Fatalf("cap(Store) = %d, want retained capacity 8", cap(ctx.store))
	}
}
