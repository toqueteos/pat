// Copyright (c) 2012 The Go Authors. All rights reserved.

package pat

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
)

// Does path match pattern?
func pathMatch(pattern, path string) bool {
	// Empty pattern matches nothing
	if len(pattern) == 0 {
		return false
	}

	if strings.Contains(pattern, ":") {
		return pathMatchSplat(pattern, path)
	}

	return pathMatchFlat(pattern, path)
}

// pathMatchFlat matches exact patterns. `path` may contain a (sub)domain.
// '/a' matches: '/a' and '/a/'
// '/a/' matches: '/a', '/a/' and '/a/whatever'
func pathMatchFlat(pattern, path string) bool {
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[:n] == pattern
}

// pathMatchSplat matches patterns with capture groups. `path` may contain a
// (sub)domain.
// '/:a' will match: '/hello' and '/hello/'
// '/:a/' will match: '/hello', '/hello/' and '/hello/whatever'
func pathMatchSplat(pattern, path string) bool {
	var leadingSlash bool

	_pattern := strings.Split(pattern, "/")
	// Number of slashes in pattern
	slashes := strings.Count(pattern, "/")

	// Patterns with a leading slash can match paths with `n-1` or more slashes,
	// `n` being the total number of slashes of `pattern`.
	if pattern[len(pattern)-1] == '/' {
		leadingSlash = true

		slashes -= 1
		// Last item of _pattern will be empty causing everything to NOT match.
		_pattern = _pattern[:slashes]
	}

	// Split path by slashes
	_path := strings.Split(path, "/")

	switch leadingSlash {
	case true:
		// Check (n-1)+ slashes on path
		if len(_path) <= slashes {
			return false
		}
	case false:
		// There should be the same number of slashes on pattern and path
		if !leadingSlash && strings.Count(path, "/") != slashes {
			return false
		}
	}

	// Traverse each path component
	for i, item := range _pattern {
		// Split by splat mark
		index := strings.Index(item, ":")

		// Determine where's the splat, if there's one
		switch index {
		// No splat found
		case -1:
			if item != _path[i] {
				return false
			}
		// Splat found
		case 0:
			// Splat will match whatever its in _path[i]
		// Prefixed splat
		default:
			prefix := item[:index]
			if !strings.HasPrefix(_path[i], prefix) {
				return false
			}
		}
	}

	return true
}

func parseSplats(pattern, path string) url.Values {
	_pattern := strings.Split(pattern, "/")
	_path := strings.Split(path, "/")

	if !strings.Contains(pattern, ":") {
		return nil
	}

	values := make(url.Values)

	// Traverse each path component
	for i, item := range _pattern {
		// Determine where's the splat, if there's one
		switch index := strings.Index(item, ":"); index {
		case -1:
		// No splat found
		case 0:
			// Splat found
			values.Add(item, _path[i])
		default:
			// Prefixed splat
			values.Add(item[index:], _path[i][index:])
		}
	}

	return values
}

// ServeMux is an HTTP request multiplexer.
// It matches the URL of each incoming request against a list of registered
// patterns and calls the handler for the pattern that
// most closely matches the URL.
//
// Patterns name fixed, rooted paths, like "/favicon.ico",
// or rooted subtrees, like "/images/" (note the trailing slash).
// Longer patterns take precedence over shorter ones, so that
// if there are handlers registered for both "/images/"
// and "/images/thumbnails/", the latter handler will be
// called for paths beginning "/images/thumbnails/" and the
// former will receive requests for any other paths in the
// "/images/" subtree.
//
// Patterns may optionally begin with a host name, restricting matches to
// URLs on that host only.  Host-specific patterns take precedence over
// general patterns, so that a handler might register for the two patterns
// "/codesearch" and "codesearch.google.com/" without also taking over
// requests for "http://www.google.com/".
//
// ServeMux also takes care of sanitizing the URL request path,
// redirecting any request containing . or .. elements to an
// equivalent .- and ..-free URL.
type ServeMux struct {
	mu sync.RWMutex
	m  map[string]muxEntry
}

type muxEntry struct {
	explicit bool
	h        http.Handler
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux { return &ServeMux{m: make(map[string]muxEntry)} }

// Return the canonical path for p, eliminating . and .. elements.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// Find a handler on a handler map given a path string
// Most-specific (longest) pattern wins
func (mux *ServeMux) match(path string) (pattern string, h http.Handler) {
	n := 0
	for k, v := range mux.m {
		if !pathMatch(k, path) {
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			pattern = k
			h = v.h
		}
	}
	return
}

// handler returns the handler to use for the request r.
func (mux *ServeMux) handler(r *http.Request) http.Handler {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	// Host-specific pattern takes precedence over generic ones
	pattern, h := mux.match(r.Host + r.URL.Path)
	if h == nil {
		pattern, h = mux.match(r.URL.Path)
	}
	if h == nil {
		h = http.NotFoundHandler()
	} else {
		params := parseSplats(pattern, r.URL.Path)
		if params != nil {
			r.URL.RawQuery = url.Values(params).Encode() + "&" + r.URL.RawQuery
		}
	}
	return h
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "CONNECT" {
		// Clean path to canonical form and redirect.
		if p := cleanPath(r.URL.Path); p != r.URL.Path {
			w.Header().Set("Location", p)
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
	}
	mux.handler(r).ServeHTTP(w, r)
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (mux *ServeMux) Handle(pattern string, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
	if handler == nil {
		panic("http: nil handler")
	}
	if mux.m[pattern].explicit {
		panic("http: multiple registrations for " + pattern)
	}

	mux.m[pattern] = muxEntry{
		explicit: true,
		h:        handler,
	}

	// Helpful behavior:
	// If pattern is /tree/, insert an implicit permanent redirect for /tree.
	// It can be overridden by an explicit registration.
	n := len(pattern)
	if n > 0 && pattern[n-1] == '/' && !mux.m[pattern[0:n-1]].explicit {
		mux.m[pattern[0:n-1]] = muxEntry{
			h: http.RedirectHandler(pattern, http.StatusMovedPermanently),
		}
	}
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler http.HandlerFunc) {
	mux.Handle(pattern, http.HandlerFunc(handler))
}
