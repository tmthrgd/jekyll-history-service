// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

// GOMAXPROCS=10 go test

package main

import (
	"bytes"
	"io"
	"testing"
)

func TestCopyBuffer(t *testing.T) {
	for i := 0; i < 2; i++ {
		rb, wb := new(bytes.Buffer), new(bytes.Buffer)
		rb.WriteString("hello, world.")

		if _, err := copyBuffer(wb, rb); err != nil {
			t.Error(err)
		}

		if wb.String() != "hello, world." {
			t.Errorf("copyBuffer did not work properly")
		}
	}
}

func TestCopyBufferParallel(t *testing.T) {
	for i := 0; i < 16; i++ {
		go TestCopyBuffer(t)
	}
}

type fakeReader int

func (fr *fakeReader) Read(p []byte) (n int, err error) {
	if int(*fr) < len(p) {
		return int(*fr), io.EOF
	}

	*(*int)(fr) -= len(p)
	return len(p), nil
}

type fakeWriter struct{}

func (fakeWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func benchmarkCopyFunc(b *testing.B, copyFn func(dst io.Writer, src io.Reader) (written int64, err error), size int) {
	r := (*fakeReader)(&size)
	w := fakeWriter{}

	for i := 0; i < b.N; i++ {
		if _, err := copyFn(w, r); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkCopyBuffer10(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 10)
}

func BenchmarkCopyBuffer1k(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 1024)
}

func BenchmarkCopyBuffer10k(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 10*1024)
}

func BenchmarkCopyBuffer100k(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 100*1024)
}

func BenchmarkCopyBuffer1m(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 1024*1024)
}

func BenchmarkCopyBuffer10m(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 10*1024*1024)
}

func BenchmarkCopyBuffer100m(b *testing.B) {
	benchmarkCopyFunc(b, copyBuffer, 100*1024*1024)
}

func BenchmarkIoCopy10(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 10)
}

func BenchmarkIoCopy1k(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 1024)
}

func BenchmarkIoCopy10k(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 10*1024)
}

func BenchmarkIoCopy100k(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 100*1024)
}

func BenchmarkIoCopy1m(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 1024*1024)
}

func BenchmarkIoCopy10m(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 10*1024*1024)
}

func BenchmarkIoCopy100m(b *testing.B) {
	benchmarkCopyFunc(b, io.Copy, 100*1024*1024)
}

func TestParsePageString(t *testing.T) {
	if n, redirect, err := parsePageString(""); n != 1 || redirect || err != nil {
		t.Errorf(`parsePageString("") = (%d, %t, %v), expected (1, false, <nil>)`, n, redirect, err)
	}

	if n, redirect, err := parsePageString("1"); n != 1 || !redirect || err != nil {
		t.Errorf(`parsePageString("1") = (%d, %t, %v), expected (1, true, <nil>)`, n, redirect, err)
	}

	if n, redirect, err := parsePageString("2"); n != 2 || redirect || err != nil {
		t.Errorf(`parsePageString("2") = (%d, %t, %v), expected (2, false, <nil>)`, n, redirect, err)
	}

	if n, redirect, err := parsePageString("5"); n != 5 || redirect || err != nil {
		t.Errorf(`parsePageString("5") = (%d, %t, %v), expected (5, false, <nil>)`, n, redirect, err)
	}

	if n, redirect, err := parsePageString("10"); n != 10 || redirect || err != nil {
		t.Errorf(`parsePageString("10") = (%d, %t, %v), expected (10, false, <nil>)`, n, redirect, err)
	}

	if n, redirect, err := parsePageString("1000"); n != 1000 || redirect || err != nil {
		t.Errorf(`parsePageString("1000") = (%d, %t, %v), expected (1000, false, <nil>)`, n, redirect, err)
	}

	if _, _, err := parsePageString("0"); err == nil {
		t.Error(`parsePageString("0") = (_, _, <nil>), expected (_, _, ...)`)
	}

	if _, _, err := parsePageString("-1"); err == nil {
		t.Error(`parsePageString("-1") = (_, _, <nil>), expected (_, _, ...)`)
	}

	if _, _, err := parsePageString("-100"); err == nil {
		t.Error(`parsePageString("-100") = (_, _, <nil>), expected (_, _, ...)`)
	}
}
