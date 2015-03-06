package zdelta

import (
	"io"
	"unsafe"
)

/*
#include "zdlib.h"
#include "zd_mem.h"
#include "zutil.h"
*/
import "C"

// Expected delta compression ratio; copied from zdelta.c
const kExpectedRatio = 4

// Maximum amount of memory to use for output buffer in streamed functions
const kDBufMaxSize = (16 * 1024)

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

/** Errors returned from the zdelta library. */
type ZDError struct {
	Status  C.int
	Message string
}

func (e *ZDError) Error() string {
	return e.Message
}
