// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
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

	if wrote, err := executeTemplate(indexTemplate, nil, w); err != nil {
		log.Printf("%[1]T %[1]v", err)

		if !wrote {
			h.Del("Cache-Control")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
