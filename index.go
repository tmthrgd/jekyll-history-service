// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

var indexCacheControl = fmt.Sprintf("public, max-age=%d", (10*time.Minute)/time.Second)

func indexHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	h := w.Header()
	h.Set("Cache-Control", indexCacheControl)

	if checkLastModified(w, r, indexModTime, 0) {
		return
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	if err := indexTemplate.Execute(buf, nil); err != nil {
		h.Del("Cache-Control")

		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	h.Set("Content-Type", "text/html; charset=utf-8")

	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("%[1]T %[1]v", err)
	}
}
