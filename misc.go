// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"sync"
)

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func copyBuffer(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := bufferPool.Get().(*bytes.Buffer)

	buf.Reset()
	buf.Grow(32 * 1024)

	written, err = io.CopyBuffer(dst, src, buf.Bytes()[:buf.Cap()])

	bufferPool.Put(buf)
	return
}

func parsePageString(page string) (int, bool, error) {
	if len(page) == 0 {
		return 1, false, nil
	}

	if num, err := strconv.Atoi(page); err != nil {
		return 0, false, err
	} else if num <= 0 {
		return 0, false, fmt.Errorf("invalid page number '%d'", num)
	} else if num == 1 {
		return 1, true, nil
	} else {
		return num, false, nil
	}
}
