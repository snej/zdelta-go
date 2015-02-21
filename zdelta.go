package zdelta

import (
	"fmt"
	"io"
	"unsafe"
)

/*
#include "zdlib.h"
#include "zd_mem.h"
#include "zutil.h"
#include <string.h>
#include <stdlib.h>
*/
import "C"

// Expected delta compression ratio; copied from zdelta.c
const kExpectedRatio = 4

// Maximum amount of memory to use for output buffer in streamed functions
const kDBufMaxSize = (256 * 1024)

// (This is a macro in zdlib.h, so I had to translate it to a function)
func zd_deflateInit(strm C.zd_streamp, level C.int) C.int {
	vers := C.CString(C.ZDLIB_VERSION)
	rval := C.zd_deflateInit_(strm, level, vers, C.int(unsafe.Sizeof(*strm)))
	C.free(unsafe.Pointer(vers))
	return rval
}

// (This is a macro in zdlib.h, so I had to translate it to a function)
func zd_inflateInit(strm C.zd_streamp) C.int {
	vers := C.CString(C.ZDLIB_VERSION)
	rval := C.zd_inflateInit_(strm, vers, C.int(unsafe.Sizeof(*strm)))
	C.free(unsafe.Pointer(vers))
	return rval
}

func arrayPtr(a []byte) *C.Bytef {
	return (*C.Bytef)(unsafe.Pointer(&a[0]))
}
func arrayLen(a []byte) C.uLong {
	return C.uLong(len(a))
}

func writeTo(strm *C.zd_stream, tbuf []byte, out io.Writer) error {
	count := uintptr(unsafe.Pointer(strm.next_out)) - uintptr(unsafe.Pointer(&tbuf[0]))
	if count > 0 {
		if _, wval := out.Write(tbuf[0:count]); wval != nil {
			return wval // Stream write error
		}
	}
	return nil
}

type Delta []byte

type ZDError C.int

func (status ZDError) Error() string {
	return fmt.Sprintf("ZDelta error %d", status)
}

/** Returns the delta from a source to a target byte array. */
func CreateDelta(src []byte, target []byte) (Delta, error) {
	var rawDelta *C.Bytef
	var deltaSize C.uLongf
	// http://cis.poly.edu/zdelta/manual.shtml#compress1
	status := C.zd_compress1(arrayPtr(src), arrayLen(src),
		arrayPtr(target), arrayLen(target),
		&rawDelta, &deltaSize)
	if status != C.ZD_OK {
		return nil, ZDError(status)
	}
	delta := make(Delta, deltaSize)
	C.memcpy(unsafe.Pointer(&delta[0]), unsafe.Pointer(rawDelta), C.size_t(deltaSize))
	C.free(unsafe.Pointer(rawDelta))
	return delta, nil
}

/** Applies a precomputed delta to a source byte array, returning the target. */
func ApplyDelta(src []byte, delta Delta) ([]byte, error) {
	var rawTarget *C.Bytef
	var targetSize C.uLongf
	// http://cis.poly.edu/zdelta/manual.shtml#compress1
	status := C.zd_uncompress1(arrayPtr(src), arrayLen(src),
		&rawTarget, &targetSize,
		arrayPtr(delta), arrayLen(delta))
	if status != C.ZD_OK {
		return nil, fmt.Errorf("ZDelta error %d", status)
	}
	target := make([]byte, targetSize)
	C.memcpy(unsafe.Pointer(&target[0]), unsafe.Pointer(rawTarget), C.size_t(targetSize))
	C.free(unsafe.Pointer(rawTarget))
	return target, nil
}

/** Given a source and a target byte array, writes the delta to the output stream. */
func WriteDelta(ref []byte, target []byte, out io.Writer) (err error) {
	// init io buffers:
	var strm C.zd_stream
	strm.base[0] = arrayPtr(ref)
	strm.base_avail[0] = arrayLen(ref)
	strm.refnum = 1

	strm.next_in = arrayPtr(target)
	strm.avail_in = C.uInt(len(target))

	// allocate the output buffer:
	dbuf_size := (C.uLong)(len(target)/kExpectedRatio + 64)
	if dbuf_size > kDBufMaxSize {
		dbuf_size = kDBufMaxSize
	}
	dbuf := make([]byte, dbuf_size)

	// init compresser:
	if rval := zd_deflateInit(&strm, C.ZD_DEFAULT_COMPRESSION); rval != C.ZD_OK {
		return ZDError(rval)
	}
	defer func() {
		if rval := C.zd_deflateEnd(&strm); rval != C.ZD_OK && err == nil {
			err = ZDError(rval)
		}
	}()

	for {
		// reset the output buffer and generate output:
		strm.next_out = arrayPtr(dbuf)
		strm.avail_out = C.uInt(dbuf_size)
		rval := C.zd_deflate(&strm, C.ZD_FINISH)
		if rval != C.ZD_OK && rval != C.ZD_STREAM_END {
			return ZDError(rval) // Compression error
		}
		// Write to the output stream:
		if err = writeTo(&strm, dbuf, out); err != nil {
			return err // Stream write error
		}
		if rval == C.ZD_STREAM_END {
			break // EOF
		}
	}
	return
}

/** Given a source and a delta, computes the target and writes it to the output stream. */
func WriteDeltaTarget(ref []byte, delta Delta, out io.Writer) (err error) {
	// init io buffers:
	var strm C.zd_stream
	strm.base[0] = arrayPtr(ref)
	strm.base_avail[0] = arrayLen(ref)
	strm.refnum = 1
	strm.avail_in = C.uInt(len(delta))
	strm.next_in = arrayPtr(delta)

	// allocate target buffer:
	tbuf_size := C.uInt(len(ref) + kExpectedRatio*len(delta))
	if tbuf_size > kDBufMaxSize {
		tbuf_size = kDBufMaxSize
	}
	tbuf := make([]byte, tbuf_size)
	strm.avail_out = tbuf_size
	strm.next_out = arrayPtr(tbuf)

	// init decompressor:
	if rval := zd_inflateInit(&strm); rval != C.ZD_OK {
		return ZDError(rval)
	}
	defer func() {
		if rval := C.zd_inflateEnd(&strm); rval != C.ZD_OK && err == nil {
			err = ZDError(rval)
		}
	}()

	for {
		// reset the output buffer and generate output:
		strm.next_out = arrayPtr(tbuf)
		strm.avail_out = tbuf_size
		rval := C.zd_inflate(&strm, C.ZD_SYNC_FLUSH)
		if rval != C.ZD_OK && rval != C.ZD_STREAM_END {
			return ZDError(rval) // Decompression error
		}
		// Write to the output stream:
		if err = writeTo(&strm, tbuf, out); err != nil {
			return err // Stream write error
		}
		if rval == C.ZD_STREAM_END {
			break // EOF
		}
	}
	return
}
