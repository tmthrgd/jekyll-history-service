// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

var repoCacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

func Repo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Cache-Control", repoCacheControl)

	if checkLastModified(w, r, time.Now(), time.Minute) {
		return
	}

	var page int = 1

	if pageStr := ps.ByName("page"); len(pageStr) != 0 {
		var err error
		if page, err = strconv.Atoi(pageStr); err != nil || page <= 0 {
			w.Header().Del("Cache-Control")

			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if page == 1 {
			localRedirect(w, r, "../../")
			return
		}
	}

	user, repo, tree := ps.ByName("user"), ps.ByName("repo"), ps.ByName("tree")

	commits, resp, err := client.Repositories.ListCommits(user, repo, &github.CommitsListOptions{
		SHA: tree,

		ListOptions: github.ListOptions{
			Page: page,

			PerPage: 50,
		},
	})
	if err != nil {
		w.Header().Del("Cache-Control")

		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	if debug {
		log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", resp.Remaining, resp.Limit, resp.Reset)
	}

	if err := repoTemplate.Execute(w, struct {
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
		w.Header().Del("Cache-Control")

		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
