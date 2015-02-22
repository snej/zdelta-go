# A Go API for zdelta

This is a [Go](http://golang.org) (golang) interface to the [zdelta](http://cis.poly.edu/zdelta/) delta-compression library.

zdelta creates and applies binary deltas (aka diffs) between arbitrary strings of bytes. 
Delta encoding is very useful as a compact representation of changes in files or data records. 
For example, all software version control systems use chains of deltas to store the history of a file over time, 
and most software update systems contain packages of deltas (patches) instead of entire files.

## Using It

This package can be directly imported into a Go source file via:
```go
import "github.com/snej/zdelta-go"
```

The zdelta source code (in C) is contained in this package, so it's standalone and you don't need to install any shared libraries into your system.  (There's no official up-to-date zdelta source repository, so I copied the source from my [unofficial](https://github.com/snej/zdelta) (but up-to-date, as far as I know) one.)

[Here's the Godoc documentation](https://godoc.org/github.com/snej/zdelta-go). And here's an example of basic usage: Let's say you have two byte arrays, `vers1` and `vers2`, containing two revisions of a file. You want to send the second version to someone who only has the first. So you generate a delta:

```go
delta, err := zdelta.CreateDelta(vers1, vers2)
```

The delta is just a byte array you can send however you like. (There are alternate APIs that directly write the delta to an output stream.) In general it will be much smaller than `vers2`, especially if the two versions are almost the same; in the worst case it could be as much as 12 bytes larger (but no more.)

On the receiving end, you just gather up `vers1` and the delta you received, and:

```go
vers2, err := zdelta.ApplyDelta(vers1, delta)
```

(Again, there's an alternate API that allows you to write the output to a stream.)

## Performance

The runtime speed should be nearly identical to the base zdelta library. Since zdelta is based on zlib, the speed of generating a delta is about the same as that of zipping (deflating) the concatenation of the two versions. I'm not so sure about the speed of applying a delta; it should be faster than unzipping the target would be, but somewhat slower than unzipping an archive the size of the delta. 

## Status

As of now (22 Feb 2015) I consider this pre-alpha. The API isn't nailed down, it hasn't been seriously used yet, and this is admittedly my first attempt at using CGo. On the other hand, the core functionality comes from a very stable C library that's been in use for over a decade.

If you find any problems or have suggestions, please file an issue here on Github. Thanks!

—Jens Alfke