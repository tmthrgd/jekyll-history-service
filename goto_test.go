// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

// GOMAXPROCS=10 go test

package main

import (
	"net/http"
	"testing"
)

type fakeResponseWriter struct{
	Headers http.Header
	Code int
}

func (rw *fakeResponseWriter) Header() http.Header {
	return rw.Headers
}

func (*fakeResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (rw *fakeResponseWriter) WriteHeader(code int) {
	rw.Code = code
}

func TestGotoHandler(t *testing.T) {
	for url, expect := range map[string]string{
		// no url
		"http://example.com/": "http://example.com/",
		"http://example.com/?url=": "http://example.com/",
		"http://example.com/?url=%3Fa=b": "http://example.com/",
		"http://example.com/?url=%23a": "http://example.com/",

		// no user, reop or commit
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2F": "http://example.com/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2F%3Fa=b": "http://example.com/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2F%23a": "http://example.com/",

		// user
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample": "http://example.com/u/example/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2F": "http://example.com/u/example/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fexample": "http://example.com/u/example/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fexample%2F": "http://example.com/u/example/",
		"http://example.com/?url=github.com%2Fexample": "http://example.com/u/example/",
		"http://example.com/?url=github.com%2Fexample%2F": "http://example.com/u/example/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%3Fa=b": "http://example.com/u/example/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%23a": "http://example.com/u/example/",

		// user + repo
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample": "http://example.com/u/example/r/example/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%2F": "http://example.com/u/example/r/example/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fexample%2Fexample": "http://example.com/u/example/r/example/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fexample%2Fexample%2F": "http://example.com/u/example/r/example/",
		"http://example.com/?url=github.com%2Fexample%2Fexample": "http://example.com/u/example/r/example/",
		"http://example.com/?url=github.com%2Fexample%2Fexample%2F": "http://example.com/u/example/r/example/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%3Fa=b": "http://example.com/u/example/r/example/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%23a": "http://example.com/u/example/r/example/",

		// user + repo + commit
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290%2F": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290%2F": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=github.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=github.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290%2F": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290%3Fa=b": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fgit%2Fgit%2Fcommit%2Fe83c5163316f89bfbde7d9ab23ca2e25604af290%23a": "http://example.com/u/git/r/git/c/e83c5163316f89bfbde7d9ab23ca2e25604af290/",

		// user + repo + tree
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%2Ftree%2Fmaster": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%2Ftree%2Fmaster%2F": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fexample%2Fexample%2Ftree%2Fmaster": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=http%3A%2F%2Fgithub.com%2Fexample%2Fexample%2Ftree%2Fmaster%2F": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=github.com%2Fexample%2Fexample%2Ftree%2Fmaster": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=github.com%2Fexample%2Fexample%2Ftree%2Fmaster%2F": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%2Ftree%2Fmaster%3Fa=b": "http://example.com/u/example/r/example/t/master/",
		"http://example.com/?url=https%3A%2F%2Fgithub.com%2Fexample%2Fexample%2Ftree%2Fmaster%23a": "http://example.com/u/example/r/example/t/master/",
	} {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Error(err)
		}

		rw := &fakeResponseWriter{make(http.Header), -1}
		gotoHandler(rw, req, nil)

		if rw.Code != http.StatusFound {
			t.Errorf("gotoHandler returned wrong status code, expteced %d, got %d", http.StatusFound, rw.Code)
		}

		if loc := rw.Headers.Get("Location"); loc != expect {
			t.Errorf("unexpected redirect, expected %s, got %s", expect, loc)
		}
	}
}
