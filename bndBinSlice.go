// Copyright 2014 Rana Ian. All rights reserved.
// Use of this source code is governed by The MIT License
// found in the accompanying LICENSE file.

package ora

/*
#include <oci.h>
#include "version.h"
*/
import "C"
import "unsafe"

type bndBinSlice struct {
	stmt   *Stmt
	ocibnd *C.OCIBind
	buf    []byte
	arrHlp
}

func (bnd *bndBinSlice) bindOra(values []Raw, position int, lobBufferSize int, stmt *Stmt, isAssocArray bool) (iterations uint32, err error) {
	binValues := make([][]byte, len(values))
	nullInds := make([]C.sb2, len(values))
	for i := range values {
		if values[i].IsNull {
			nullInds[i] = C.sb2(-1)
		} else {
			binValues[i] = values[i].Value
		}
	}
	return bnd.bind(binValues, nullInds, position, lobBufferSize, stmt, isAssocArray)
}

func (bnd *bndBinSlice) bind(values [][]byte, nullInds []C.sb2, position int, lobBufferSize int, stmt *Stmt, isAssocArray bool) (iterations uint32, err error) {
	bnd.stmt = stmt
	L, C := len(values), cap(values)
	iterations, curlenp, needAppend := bnd.ensureBindArrLength(&L, &C, isAssocArray)
	if needAppend {
		values = append(values, []byte{})
	}
	var maxLen int
	for _, b := range values {
		if len(b) > maxLen {
			maxLen = len(b)
		}
	}
	n := maxLen * L
	if cap(bnd.buf) < n {
		//bnd.buf = make([]byte, n)
		bnd.buf = bytesPool.Get(n)[:n]
	} else {
		bnd.buf = bnd.buf[:n]
		// reset buffer
		for i := range bnd.buf {
			bnd.buf[i] = 0
		}
	}
	for i, b := range values {
		copy(bnd.buf[i*maxLen:], b)
		bnd.alen[i] = C.ACTUAL_LENGTH_TYPE(len(b))
	}
	r := C.OCIBINDBYPOS(
		bnd.stmt.ocistmt,                 //OCIStmt      *stmtp,
		&bnd.ocibnd,                      //OCIBind      **bindpp,
		bnd.stmt.ses.srv.env.ocierr,      //OCIError     *errhp,
		C.ub4(position),                  //ub4          position,
		unsafe.Pointer(&bnd.buf[0]),      //void         *valuep,
		C.LENGTH_TYPE(maxLen),            //sb8          value_sz,
		C.SQLT_LBI,                       //ub2          dty,
		unsafe.Pointer(&bnd.nullInds[0]), //void         *indp,
		&bnd.alen[0],                     //ub4          *alenp,
		&bnd.rcode[0],                    //ub2          *rcodep,
		getMaxarrLen(C, isAssocArray),    //ub4          maxarr_len,
		curlenp,       //ub4          *curelep,
		C.OCI_DEFAULT) //ub4          mode );
	if r == C.OCI_ERROR {
		return iterations, bnd.stmt.ses.srv.env.ociError()
	}
	r = C.OCIBindArrayOfStruct(
		bnd.ocibnd,
		bnd.stmt.ses.srv.env.ocierr,
		C.ub4(maxLen),       //ub4         pvskip,
		C.ub4(C.sizeof_sb2), //ub4         indskip,
		C.ub4(C.sizeof_ub4), //ub4         alskip,
		C.ub4(C.sizeof_ub2)) //ub4         rcskip
	if r == C.OCI_ERROR {
		return iterations, bnd.stmt.ses.srv.env.ociError()
	}
	return iterations, nil
}

func (bnd *bndBinSlice) setPtr() error {
	return nil
}

func (bnd *bndBinSlice) close() (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = errR(value)
		}
	}()
	stmt := bnd.stmt
	bnd.stmt = nil
	bnd.ocibnd = nil
	bytesPool.Put(bnd.buf)
	bnd.buf = nil
	bnd.arrHlp.close()
	stmt.putBnd(bndIdxBinSlice, bnd)
	return nil
}
