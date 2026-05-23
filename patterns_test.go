package router

import "testing"

func TestIsLowerAlpha(t *testing.T) {
	ok := []string{"abc", "lowercase", "xyz"}
	fail := []string{"ABC", "123", "a1", "a_b", "", "lower case"}

	runMatcherTest(t, isLowerAlpha, ok, fail)
}

func TestIsUpperAlpha(t *testing.T) {
	ok := []string{"ABC", "UPPER", "XYZ"}
	fail := []string{"abc", "123", "A1", "", "UP PER"}

	runMatcherTest(t, isUpperAlpha, ok, fail)
}

func TestIsAlpha(t *testing.T) {
	ok := []string{"abc", "XYZ", "MiXeD"}
	fail := []string{"abc123", "123", "a_b", "hello-world", ""}

	runMatcherTest(t, isAlpha, ok, fail)
}

func TestIsDigits(t *testing.T) {
	ok := []string{"0", "123", "456789"}
	fail := []string{"abc", "12a", "", " 1", "1 "}

	runMatcherTest(t, isDigits, ok, fail)
}

func TestIsAlnum_TableBased(t *testing.T) {
	ok := []string{
		"0",
		"1",
		"01",
		"ab",
		"AB",
		"ab10",
		"ba01",
		"abc",
		"A1b2",
		"Z9x8",
	}
	fail := []string{
		"",
		"a_b",
		"10-",
		"a b",
		"hello-world",
	}

	runMatcherTest(t, isAlnum, ok, fail)
}

func TestIsWord(t *testing.T) {
	ok := []string{"abc", "ABC123", "user_1", "x9"}
	fail := []string{"", "dash-separated", "with space", "emoji🙂"}

	runMatcherTest(t, isWord, ok, fail)
}

func TestIsSlugSafe(t *testing.T) {
	ok := []string{"abc-123", "user_name-1", "ABC-xyz", "a_b-c_d"}
	fail := []string{"", "has space", "slash/inside", "dot.not.allowed?"}

	runMatcherTest(t, isSlugSafe, ok, fail)
}

func TestIsSlug(t *testing.T) {
	ok := []string{"abc", "abc-123", "abc123", "a-b-c", "slug-test", "0-1-2"}
	fail := []string{"", "ABC", "User_Name", "has space", "with.dot", "slash/inside"}

	runMatcherTest(t, isSlug, ok, fail)
}

func TestIsHex(t *testing.T) {
	ok := []string{"0", "1a2B", "deadBEEF", "CAFEBABE"}
	fail := []string{"", "xyz", "123g", "+++", "g123"}

	runMatcherTest(t, isHex, ok, fail)
}

func TestIsUUID(t *testing.T) {
	ok := []string{"12345678-1234-1234-1234-1234567890ab"}
	fail := []string{
		"",
		"12345678-1234-1234-1234",
		"12345678-1234-1234-1234-1234567890",
		"g2345678-1234-1234-1234-1234567890ab",
		"123456781234-1234-1234-1234567890ab",
	}

	runMatcherTest(t, isUUID, ok, fail)
}

func TestIsSafeText(t *testing.T) {
	ok := []string{
		"Hello world",
		"file-name_1",
		"Name_With-Dash.",
		"NoSeparator",
	}
	fail := []string{
		"",
		"bad!",
		"with,comma",
		"slash/inside",
	}

	runMatcherTest(t, isSafeText, ok, fail)
}

func TestIsUpperAlnum(t *testing.T) {
	ok := []string{"ABC", "ABC123", "Z9"}
	fail := []string{"", "abc", "AbC", "A_B", "A-b"}

	runMatcherTest(t, isUpperAlnum, ok, fail)
}

func TestIsBase64(t *testing.T) {
	ok := []string{
		"TWFu",
		"abcdEFGH123+/",
		"====",
	}
	fail := []string{
		"",
		"abc-",
		"hello world",
		"*",
	}

	runMatcherTest(t, isBase64, ok, fail)
}

func TestIsDateYMD(t *testing.T) {
	ok := []string{
		"2024-01-02",
		"1999-12-31",
		"0000-00-00",
	}
	fail := []string{
		"",
		"2024-1-2",
		"20240102",
		"2024/01/02",
		"abcd-ef-gh",
	}

	runMatcherTest(t, isDateYMD, ok, fail)
}

func TestIsSafePath(t *testing.T) {
	ok := []string{
		"file.txt",
		"path/to/file.txt",
		"A_B-1/2",
		"dir/sub_dir-01.file",
	}
	fail := []string{
		"",
		"path with space",
		"path?query",
		"path#fragment",
	}

	runMatcherTest(t, isSafePath, ok, fail)
}

func TestIsAnyAndAlwaysTrue(t *testing.T) {
	inputs := []string{"", "anything", "123", "/weird/path"}

	for _, in := range inputs {
		if !isAny(in) {
			t.Errorf("isAny(%q) = false, want true", in)
		}
		if !alwaysTrue(in) {
			t.Errorf("alwaysTrue(%q) = false, want true", in)
		}
	}
}

func TestFunctionMatchersContainsExpectedKeys(t *testing.T) {
	expected := []string{
		"isLowerAlpha",
		"isUpperAlpha",
		"isAlpha",
		"isDigits",
		"isAlnum",
		"isWord",
		"isSlugSafe",
		"isSlug",
		"isHex",
		"isUUID",
		"isSafeText",
		"isUpperAlnum",
		"isBase64",
		"isDateYMD",
		"isSafePath",
		"any",
	}

	for _, key := range expected {
		fn, ok := functionMatchers[key]
		if !ok {
			t.Fatalf("functionMatchers missing key %q", key)
		}
		if !fn("test") && key == "any" {
			t.Errorf("functionMatchers[%q](\"test\") = false, want true", key)
		}
	}
}

func TestPatternMatchersBasicMappings(t *testing.T) {
	tests := []struct {
		pattern    string
		shouldPass []string
		shouldFail []string
	}{
		{
			pattern:    `[a-z]+`,
			shouldPass: []string{"abc"},
			shouldFail: []string{"ABC", "123", ""},
		},
		{
			pattern:    `[A-Z]+`,
			shouldPass: []string{"ABC"},
			shouldFail: []string{"abc", "123", ""},
		},
		{
			pattern:    `[0-9]+`,
			shouldPass: []string{"123"},
			shouldFail: []string{"abc", "12a", ""},
		},
		{
			pattern:    `\w+`,
			shouldPass: []string{"abc123", "User_1"},
			shouldFail: []string{"has space", "", "-"},
		},
		{
			pattern:    `[a-z0-9\-]+`,
			shouldPass: []string{"abc-123", "slug-test"},
			shouldFail: []string{"ABC", "slug_test", "", "space here"},
		},
		{
			pattern:    `[a-fA-F0-9]+`,
			shouldPass: []string{"deadBEEF", "0F1a"},
			shouldFail: []string{"xyz", "g123", ""},
		},
		{
			pattern:    `8-4-4-4-12`,
			shouldPass: []string{"12345678-1234-1234-1234-1234567890ab"},
			shouldFail: []string{"not-a-uuid", "12345678-1234-1234-1234"},
		},
		{
			pattern:    `.*`,
			shouldPass: []string{"", "anything", "123", "/path"},
			shouldFail: nil,
		},
	}

	for _, tt := range tests {
		fn, ok := patternMatchers[tt.pattern]
		if !ok {
			t.Fatalf("patternMatchers missing pattern %q", tt.pattern)
		}

		for _, s := range tt.shouldPass {
			if !fn(s) {
				t.Errorf("Pattern %q: expected %q to match, but it did not", tt.pattern, s)
			}
		}
		for _, s := range tt.shouldFail {
			if fn(s) {
				t.Errorf("Pattern %q: expected %q to NOT match, but it did", tt.pattern, s)
			}
		}
	}
}

func runMatcherTest(t *testing.T, fn matchFunc, shouldMatch, shouldNotMatch []string) {
	for _, input := range shouldMatch {
		if !fn(input) {
			t.Errorf("Expected %q to match but it did not", input)
		}
	}
	for _, input := range shouldNotMatch {
		if fn(input) {
			t.Errorf("Expected %q to not match but it did", input)
		}
	}
}
