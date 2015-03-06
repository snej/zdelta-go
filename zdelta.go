/*
Package zdelta is a Go interface to the zdelta library (http://cis.poly.edu/zdelta/),
which creates and applies binary deltas (aka diffs) between arbitrary strings of bytes.

Delta encoding is very useful as a compact
representation of changes in files or data records. For example, all software version
control systems use chains of deltas to store the history of a file over time, and most
software update systems distribute deltas (patches) instead of entire files.

Zdelta is based on the Deflate algorithm, and its implementation is a modified version of
the zlib library. Creating a delta from Source to Target is conceptually like running
Source through deflate(), throwing away the output so far, then running Target through the
same deflate instance.
*/
package zdelta

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"unsafe"
)

/*
#include "zdlib.h"
#include "zd_mem.h"
#include "zutil.h"
*/
import "C"

type codec struct {
	strm C.zd_stream
	buf  []byte
}

func (c *codec) setupStrm(ref []byte, other []byte) {
	c.strm.base[0] = arrayPtr(ref)
	c.strm.base_avail[0] = arrayLen(ref)
	c.strm.base_out[0] = 0
	c.strm.refnum = 1
	c.strm.next_in = arrayPtr(other)
	c.strm.avail_in = C.uInt(len(other))
	c.strm.total_in = 0
	c.strm.total_out = 0
}

func (c *codec) allocbuf(buf_size int) {
	if c.buf == nil {
		if buf_size > kDBufMaxSize {
			buf_size = kDBufMaxSize
		}
		c.buf = make([]byte, buf_size)
	}
}

func (c *codec) resetBuf() {
	c.strm.next_out = arrayPtr(c.buf)
	c.strm.avail_out = C.uInt(len(c.buf))
}

func (c *codec) writeBuf(out io.Writer) error {
	count := uintptr(unsafe.Pointer(c.strm.next_out)) - uintptr(unsafe.Pointer(&c.buf[0]))
	if count > 0 {
		if _, wval := out.Write(c.buf[0:count]); wval != nil {
			return wval // Stream write error
		}
	}
	return nil
}

func (c *codec) mkError(status C.int) error {
	err := ZDError{Status: status}
	if c.strm.msg != nil {
		err.Message = C.GoString(c.strm.msg)
	} else {
		err.Message = fmt.Sprintf("Zdelta error %d", status)
	}
	return &err
}

// An object that creates deltas. Can be reused, but not on multiple threads at once.
// Reusing a Compressor is more memory-efficient than calling the CreateDelta function many
// times in a row.
type Compressor struct {
	codec
}

// An object that applies deltas. Can be reused, but not on multiple threads at once.
// Reusing a Decompressor is more memory-efficient than calling the ApplyDelta function many
// times in a row.
type Decompressor struct {
	codec
}

//////// COMPRESSOR:

// NewCompressor creates a Compressor with a specified buffer size.
//
// You could also just use an uninitialized Compressor struct, in which case a default
// size buffer will be created.
func NewCompressor(buf_size uint) Compressor {
	var c Compressor
	c.buf = make([]byte, buf_size)
	return c
}

// NewCompressor creates a Decompressor with a specified buffer size.
//
// You could also just use an uninitialized Decompressor struct, in which case a default
// size buffer will be created.
func NewDecompressor(buf_size uint) Decompressor {
	var c Decompressor
	c.buf = make([]byte, buf_size)
	return c
}

// Given a source and a target byte array, writes the delta to the output stream.
//
// If WriteTarget or ApplyDelta is later called with the same source and delta,
// it will output the same target.
func (c *Compressor) WriteDelta(source []byte, target []byte, out io.Writer) (err error) {
	c.setupStrm(source, target)
	c.allocbuf(len(target)/kExpectedRatio + 64)

	// init compresser:
	if status := zd_deflateInit(&c.strm, C.ZD_DEFAULT_COMPRESSION); status != C.ZD_OK {
		return c.mkError(status)
	}
	defer func() { C.zd_deflateEnd(&c.strm) }()

	for {
		// empty the output buffer and generate output:
		c.resetBuf()
		status := C.zd_deflate(&c.strm, C.ZD_FINISH)
		if status != C.ZD_OK && status != C.ZD_STREAM_END {
			return c.mkError(status) // Compression error
		}
		// Write the buffer to the output stream:
		if err = c.writeBuf(out); err != nil {
			return err // Stream write error
		}
		if status == C.ZD_STREAM_END {
			break // EOF
		}
	}
	return
}

// Given a source and a target byte array, returns the delta.
//
// If WriteTarget or ApplyDelta is later called with the same source and delta,
// it will output the same target.
func (c *Compressor) CreateDelta(src []byte, target []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := c.WriteDelta(src, target, &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

//////// DECOMPRESSOR:

// ApplyDelta applies a precomputed delta to a source byte array, and writes the target
// data to a Writer.
func (d *Decompressor) WriteTarget(source []byte, delta []byte, out io.Writer) (err error) {
	d.setupStrm(source, delta)
	d.allocbuf(len(source) + kExpectedRatio*len(delta))

	// init decompressor:
	if status := zd_inflateInit(&d.strm); status != C.ZD_OK {
		return d.mkError(status)
	}
	defer func() { C.zd_inflateEnd(&d.strm) }()

	for {
		// reset the output buffer and generate output:
		d.resetBuf()
		rval := C.zd_inflate(&d.strm, C.ZD_SYNC_FLUSH)
		if rval != C.ZD_OK && rval != C.ZD_STREAM_END {
			return d.mkError(rval) // Decompression error
		}
		// Write to the output stream:
		if err = writeTo(&d.strm, d.buf, out); err != nil {
			return err // Stream write error
		}
		if rval == C.ZD_STREAM_END {
			break // EOF
		}
	}
	return
}

// ApplyDelta applies a precomputed delta to a source byte array, returning the target.
func (d *Decompressor) ApplyDelta(src []byte, delta []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := d.WriteTarget(src, delta, &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

//////// CONVENIENCES:

var sCompressorPool, sDecompressorPool sync.Pool

func init() {
	sCompressorPool.New = func() interface{} {
		c := NewCompressor(kDBufMaxSize)
		return &c
	}
	sDecompressorPool.New = func() interface{} {
		d := NewDecompressor(kDBufMaxSize)
		return &d
	}
}

// CreateDelta is a convenience function that calls Compressor.CreateDelta with a temporary
// Compressor.
func CreateDelta(src []byte, target []byte) ([]byte, error) {
	c := sCompressorPool.Get().(*Compressor)
	result, err := c.CreateDelta(src, target)
	sCompressorPool.Put(c)
	return result, err
}

// ApplyDelta is a convenience function that calls Decompressor.ApplyDelta with a temporary
// Decompressor.
func ApplyDelta(src []byte, delta []byte) ([]byte, error) {
	d := sDecompressorPool.Get().(*Decompressor)
	result, err := d.ApplyDelta(src, delta)
	sDecompressorPool.Put(d)
	return result, err
}
