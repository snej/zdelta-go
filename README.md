# A Go API for zdelta
This is a [Go](http://golang.org) (golang) interface to the [zdelta](http://cis.poly.edu/zdelta/) delta-compression library.

zdelta creates and applies binary deltas (aka diffs) between arbitrary strings of bytes. 
Delta encoding is very useful as a compact representation of changes in files or data records. 
For example, all software version control systems use chains of deltas to store the history of a file over time, 
and most software update systems contain packages of deltas (patches) instead of entire files.

This package can be directly imported into a Go source file via:
```go
import "github.com/snej/zdelta-go"
```
