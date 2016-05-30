// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
)

type buildJekyllGetter struct {
	RepoBasePath string
	SiteBasePath string

	Client *http.Client
}

func (bj buildJekyllGetter) Get(_ groupcache.Context, key string, dest groupcache.Sink) error {
	parts := strings.Split(key, "\x00")
	if len(parts) != 4 {
		return &httpError{errors.New("invalid key"), http.StatusBadRequest}
	}

	tag, user, repo, commit := parts[0], parts[1], parts[2], parts[3]

	repoPath := filepath.Join(bj.RepoBasePath, tag[0:1], tag[1:2], tag[2:])
	sitePath := filepath.Join(bj.SiteBasePath, tag[0:1], tag[1:2], tag[2:])

	if _, err := os.Stat(sitePath); err == nil {
		return nil
	}

	if _, err := os.Stat(repoPath); err != nil {
		u, gresp, err := client.Repositories.GetArchiveLink(user, repo, github.Tarball, &github.RepositoryContentGetOptions{
			Ref: commit,
		})
		if err != nil {
			return &httpError{err, http.StatusBadGateway}
		}

		if debug {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", gresp.Remaining, gresp.Limit, gresp.Reset)
		}

		if u == nil {
			return os.ErrNotExist
		}

		client := bj.Client
		if client == nil {
			client = http.DefaultClient
		}

		resp, err := client.Do(&http.Request{
			URL:  u,
			Host: u.Host,
			Header: http.Header{
				"User-Agent": []string{fullVersionStr},
			},
		})
		if err != nil {
			return &httpError{err, http.StatusBadGateway}
		}

		if resp.Body == nil {
			return errors.New("(*http.Client).Do did not return body")
		}

		defer resp.Body.Close()

		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}

		defer reader.Close()

		if !debug {
			defer os.RemoveAll(repoPath)
		}

		tarReader := tar.NewReader(reader)

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			idx := strings.IndexRune(header.Name, filepath.Separator)
			if idx == -1 {
				continue
			}

			path := filepath.Join(repoPath, header.Name[idx+1:])

			info := header.FileInfo()
			mode := info.Mode()

			if info.IsDir() {
				if err = os.MkdirAll(path, mode); err != nil {
					return err
				}

				continue
			}

			if mode&(os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
				log.Printf("tar file '%s' has invalid mode: %d", header.Name, mode)
				continue
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}

			_, err = io.Copy(file, tarReader)
			file.Close()

			if err != nil {
				return err
			}
		}
	}

	cmd := exec.Command("jekyll", "build", "--no-watch", "--quiet", "--safe", "-s", repoPath, "-d", sitePath)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

var buildJekyll *groupcache.Group
