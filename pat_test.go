package pat

import "testing"

type PathTable struct {
	pattern, path string
	expected      bool
}

// (*) = Both patterns get auto-registered '/hello' and '/hello/' but the one
// with a leading slash doesn't match.

var pathTests = []PathTable{
	{"/", "", false},                              // 1
	{"/", "/", true},                              // 2
	{"/", "/hello", true},                         // 3
	{"/hello/", "/hello", false},                  // 4 (*)
	{"/hello/", "/helloo", false},                 // 5
	{"/hello/", "/hello/", true},                  // 6
	{"/hello/", "/hello/whatever", true},          // 7
	{"/:a", "/hello", true},                       // 8
	{"/:a", "/hello/", false},                     // 9
	{"/:a", "/world", true},                       // 10
	{"/:a", "/hello/world", false},                // 11
	{"/:a/", "/hello/world", true},                // 12
	{"/:a/", "/hello/world/world", true},          // 13
	{"/hello/:a", "/hello", false},                // 14
	{"/hello/:a", "/hello/", true},                // 15
	{"/hello/:a", "/helloo/", false},              // 16
	{"/hello/:a", "/hello/world", true},           // 17
	{"/hello/:a/", "/hello/world/whatever", true}, // 18
}

func TestMatchPath(t *testing.T) {
	for i, item := range pathTests {
		output := pathMatch(item.pattern, item.path)
		if output != item.expected {
			t.Errorf("%d. match(%q, %q) => %v, want %v", i, item.pattern, item.path, output, item.expected)
		}
	}
}
