// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

func getCommitHandler(githubClient *github.Client, highlightStyle string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var cacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Cache-Control", cacheControl)

		if checkLastModified(w, r, time.Now(), time.Minute) {
			return
		}

		user, repo, commit := ps.ByName("user"), ps.ByName("repo"), ps.ByName("commit")

		repoCommit, resp, err := githubClient.Repositories.GetCommit(user, repo, commit)
		if err != nil {
			w.Header().Del("Cache-Control")

			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		if debug {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", resp.Remaining, resp.Limit, resp.Reset)
		}

		if err := commitTemplate.Execute(w, struct {
			User   string
			Repo   string
			Commit *github.RepositoryCommit

			HighlightStyle string
		}{
			User:   user,
			Repo:   repo,
			Commit: repoCommit,

			HighlightStyle: highlightStyle,
		}); err != nil {
			w.Header().Del("Cache-Control")

			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func getBuildCommitHandler(buildJekyll *groupcache.Group) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Cache-Control", "max-age=0")

		user, repo, commit := ps.ByName("user"), ps.ByName("repo"), ps.ByName("commit")

		data := user + "\x00" + repo + "\x00" + commit

		rawTag := sha256.Sum256([]byte(data))
		tag := hex.EncodeToString(rawTag[:16])

		var res []byte

		if err := buildJekyll.Get(nil, tag+"\x00"+data, groupcache.TruncatingByteSliceSink(&res)); err != nil {
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

		url := *r.URL
		url.Host = tag + ".jekyllhistory.org"

		if _, port, err := net.SplitHostPort(r.Host); err == nil {
			url.Host = net.JoinHostPort(url.Host, port)
		}

		url.Path = ps.ByName("path")
		http.Redirect(w, r, url.String(), http.StatusFound)
	}
}
