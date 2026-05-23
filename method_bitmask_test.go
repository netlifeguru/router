package router

import (
	"testing"
)

type dummyRouter struct {
	Router
}

func TestGetMethodIndex(t *testing.T) {
	r := &dummyRouter{}

	tests := map[string]int{
		"GET":     GET,
		"POST":    POST,
		"PUT":     PUT,
		"DELETE":  DELETE,
		"PATCH":   PATCH,
		"HEAD":    HEAD,
		"OPTIONS": OPTIONS,
		"ANY":     ANY,
		"INVALID": -512,
	}

	for method, expected := range tests {
		got := r.getMethodIndex(method)
		if got != expected {
			t.Errorf("getMethodIndex(%q) = %d, want %d", method, got, expected)
		}
	}
}

func TestIndexToBit(t *testing.T) {
	r := &dummyRouter{}
	tests := map[int]int{
		GET:     0,
		POST:    1,
		PUT:     2,
		DELETE:  3,
		PATCH:   4,
		HEAD:    5,
		OPTIONS: 6,
		CONNECT: 7,
		TRACE:   8,
		ANY:     9,
	}

	for input, expected := range tests {
		got := r.indexToBit(input)
		if got != expected {
			t.Errorf("indexToBit(%d) = %d, want %d", input, got, expected)
		}
	}
}

func TestGetBitmaskIndex(t *testing.T) {
	r := &dummyRouter{}
	tests := map[string]int{
		"GET":     1,
		"POST":    2,
		"PUT":     4,
		"DELETE":  8,
		"PATCH":   16,
		"HEAD":    32,
		"OPTIONS": 64,
		"CONNECT": 128,
		"TRACE":   256,
		"UNKNOWN": 0,
	}

	for method, expected := range tests {
		got := r.getBitmaskIndex(method)
		if got != expected {
			t.Errorf("getBitmaskIndex(%q) = %d, want %d", method, got, expected)
		}
	}
}

func TestMethodsToBitmask(t *testing.T) {
	r := &dummyRouter{}

	tests := []struct {
		input    string
		expected int
	}{
		{"GET ", GET},
		{"GET POST ", GET | POST},
		{"PUT DELETE POST ", PUT | DELETE | POST},
		{"INVALID ", -1},
		{"", 0},
	}

	for _, tt := range tests {
		result := r.methodsToBitmask(tt.input)
		if result != tt.expected {
			t.Errorf("MethodsToBitmask(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestMethodsToBitmask_AnyAlone(t *testing.T) {
	r := &dummyRouter{}

	got := r.methodsToBitmask("ANY ")
	if got != 511 {
		t.Errorf("MethodsToBitmask(%q) = %d, want %d", "ANY ", got, 511)
	}
}

func TestMethodsToBitmask_AnyWithOthers(t *testing.T) {
	r := &dummyRouter{}

	got := r.methodsToBitmask("GET ANY POST ")
	if got != 511 {
		t.Errorf("MethodsToBitmask(%q) = %d, want %d", "GET ANY POST ", got, 511)
	}
}

func TestMethodsToBitmask_UnknownMixed(t *testing.T) {
	r := &dummyRouter{}

	got := r.methodsToBitmask("GET FOO ")
	if got != -1 {
		t.Errorf("MethodsToBitmask(%q) = %d, want %d", "GET FOO ", got, -1)
	}
}

func TestMethodsToBitmask_NoTrailingSpace(t *testing.T) {
	r := &dummyRouter{}

	got := r.methodsToBitmask("GET POST")
	expected := GET | POST
	if got != expected {
		t.Errorf("MethodsToBitmask(%q) = %d, want %d", "GET POST", got, expected)
	}
}
