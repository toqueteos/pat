pat
===

Improved std http.ServeMux. It supports [pat-like](http://www.godoc.org/github.com/bmizerany/pat) patterns but longer patterns take precedence over shorter ones.

Previously located at https://github.com/toqueteos/web/pat.

Usage
=====

```go
package main

import (
    "fmt"
    "github.com/toqueteos/pat"
    "net/http"
)

func Hello(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get(":name")
    fmt.Fprintf(w, "Hello %s!", name)
}

func main() {
    r := pat.NewServeMux()
    r.HandleFunc("/hello/:name", Hello)
    http.Handle("/", r)

    http.ListenAndServe(":8000", nil)
}
```

**What do parametrized URLs catch?**

Patterns will match routes the same exact way as the std http.ServeMux does, but params will only catch text until the next slash.

Check out [pat_test.go](https://github.com/toqueteos/web/blob/master/pat/pat_test.go) for all possible combinations.
