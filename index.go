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
	"time"

	"github.com/julienschmidt/httprouter"
)

var indexCacheControl = fmt.Sprintf("public, max-age=%d", (10*time.Minute)/time.Second)

func indexHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Cache-Control", indexCacheControl)

	if checkLastModified(w, r, indexModTime, 0) {
		return
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	if err := indexTemplate.Execute(buf, nil); err != nil {
		w.Header().Del("Cache-Control")

		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("%[1]T %[1]v", err)
	}
}
