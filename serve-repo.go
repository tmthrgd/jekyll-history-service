// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang/groupcache"
)

var (
	builtRepoCacheControl = fmt.Sprintf("public, max-age=%d", (10*365*24*time.Hour)/time.Second)

	timeZero time.Time

	hostRegex = regexp.MustCompile(`^([0-9a-fA-F]{32}).jekyllhistory.org$`)
)

type repoSwitch struct {
	BuiltFiles *groupcache.Group
}

func (rs repoSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	m := hostRegex.FindStringSubmatch(host)
	if m == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	tag := strings.ToLower(m[1])

	switch r.Method {
	case http.MethodGet:
	case http.MethodHead:
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead+", "+http.MethodOptions)
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/index.html") {
		localRedirect(w, r, "./")
		return
	}

	h := w.Header()
	h.Set("Cache-Control", builtRepoCacheControl)
	h.Set("Etag", `"`+tag+`"`)

	if checkETag(w, r) {
		return
	}

	var dst []byte

	if err := rs.BuiltFiles.Get(nil, tag+"\x00"+r.URL.Path, groupcache.AllocatingByteSliceSink(&dst)); err != nil {
		if nf, ok := err.(notFoundError); ok {
			http.ServeContent(w, r, r.URL.Path, timeZero, bytes.NewReader([]byte(nf)))
			return
		}

		h.Del("Cache-Control")
		h.Del("Etag")

		if herr, ok := err.(*httpError); ok {
			log.Printf("%[1]T %[1]v", herr.Err)
			http.Error(w, http.StatusText(herr.Code), herr.Code)
		} else if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	http.ServeContent(w, r, r.URL.Path, timeZero, bytes.NewReader(dst))
}
