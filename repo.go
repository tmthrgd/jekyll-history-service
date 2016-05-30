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

	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

func getRepoHandler(githubClient *github.Client) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var cacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h := w.Header()
		h.Set("Cache-Control", cacheControl)

		if checkLastModified(w, r, time.Now(), time.Minute) {
			return
		}

		page, redirect, err := parsePageString(ps.ByName("page"))
		if err != nil {
			h.Del("Cache-Control")

			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if redirect {
			localRedirect(w, r, "../../")
			return
		}

		user, repo, tree := ps.ByName("user"), ps.ByName("repo"), ps.ByName("tree")

		commits, resp, err := githubClient.Repositories.ListCommits(user, repo, &github.CommitsListOptions{
			SHA: tree,

			ListOptions: github.ListOptions{
				Page: page,

				PerPage: 50,
			},
		})
		if err != nil {
			h.Del("Cache-Control")

			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		if debug {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", resp.Remaining, resp.Limit, resp.Reset)
		}

		buf := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		if err := repoTemplate.Execute(buf, struct {
			User    string
			Repo    string
			Tree    string
			Commits []github.RepositoryCommit
			Resp    *github.Response
		}{
			User:    user,
			Repo:    repo,
			Tree:    tree,
			Commits: commits,
			Resp:    resp,
		}); err != nil {
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
}