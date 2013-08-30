// pat is a HTTP request multiplexer based on Go's std http.ServeMux with pat-
// like routes support, longer patterns take precedence over shorter ones.
// Patterns may optionally begin with a host name, restricting matches to URLs
// on that host only.
package pat
