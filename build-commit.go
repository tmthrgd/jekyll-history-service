// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/golang/groupcache"
	"github.com/julienschmidt/httprouter"
)

func getBuildCommitHandler(buildJekyll *groupcache.Group) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Cache-Control", "max-age=0")

		user, repo, commit := ps.ByName("user"), ps.ByName("repo"), ps.ByName("commit")

		data := user + "\x00" + repo + "\x00" + commit

		rawTag := sha256.Sum256([]byte(data))
		tag := hex.EncodeToString(rawTag[:16])

		var resp BuildJekyllResponse

		if err := buildJekyll.Get(nil, tag+"\x00"+data, groupcache.ProtoSink(&resp)); err != nil {
			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if len(resp.Error) != 0 {
			switch resp.Code {
			case http.StatusNotFound:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			case 0:
				log.Println(resp.Error)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			default:
				log.Println(resp.Error)
				http.Error(w, http.StatusText(int(resp.Code)), int(resp.Code))
			}

			return
		}

		url := *r.URL
		url.Host = tag + "." + r.Host
		url.Path = ps.ByName("path")

		http.Redirect(w, r, url.String(), http.StatusFound)
	}
}
