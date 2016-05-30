// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
)

type buildJekyllGetter struct {
	RepoBasePath string
	SiteBasePath string

	GithubClient *github.Client
	HTTPClient   *http.Client
}

func (bj buildJekyllGetter) Get(_ groupcache.Context, key string, dest groupcache.Sink) error {
	var resp BuildJekyllResponse

	parts := strings.Split(key, "\x00")
	if len(parts) != 4 {
		resp.Error = "invalid key"
		resp.Code = http.StatusBadRequest
		return dest.SetProto(&resp)
	}

	tag, user, repo, commit := parts[0], parts[1], parts[2], parts[3]

	repoPath := filepath.Join(bj.RepoBasePath, tag[0:1], tag[1:2], tag[2:])
	sitePath := filepath.Join(bj.SiteBasePath, tag[0:1], tag[1:2], tag[2:])

	if _, err := os.Stat(sitePath); err == nil {
		return dest.SetProto(&resp)
	}

	if _, err := os.Stat(repoPath); err != nil {
		u, gresp, err := bj.GithubClient.Repositories.GetArchiveLink(user, repo, github.Tarball, &github.RepositoryContentGetOptions{
			Ref: commit,
		})
		if err != nil {
			resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
			resp.Code = http.StatusBadGateway
			return dest.SetProto(&resp)
		}

		if debug {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", gresp.Remaining, gresp.Limit, gresp.Reset)
		}

		if u == nil {
			resp.Error = "not found"
			resp.Code = http.StatusNotFound
			return dest.SetProto(&resp)
		}

		client := bj.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}

		hresp, err := client.Do(&http.Request{
			URL:  u,
			Host: u.Host,
			Header: http.Header{
				"User-Agent": []string{fullVersionStr},
			},
		})
		if err != nil {
			resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
			resp.Code = http.StatusBadGateway
			return dest.SetProto(&resp)
		}

		if hresp.Body == nil {
			resp.Error = "(*http.Client).Do did not return body"
			return dest.SetProto(&resp)
		}

		defer hresp.Body.Close()

		reader, err := gzip.NewReader(hresp.Body)
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
				resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
				return dest.SetProto(&resp)
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
					resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
					return dest.SetProto(&resp)
				}

				continue
			}

			if mode&(os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
				log.Printf("tar file '%s' has invalid mode: %d", header.Name, mode)
				continue
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
				return dest.SetProto(&resp)
			}

			_, err = io.Copy(file, tarReader)
			file.Close()

			if err != nil {
				resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
				return dest.SetProto(&resp)
			}

			if !header.ModTime.IsZero() && !header.ModTime.Equal(unixEpochTime) {
				access := header.AccessTime

				if access.IsZero() || access.Equal(unixEpochTime) {
					access = time.Now()
				}

				if err := os.Chtimes(path, access, header.ModTime); err != nil {
					log.Printf("%[1]T: %[1]v", err)
				}
			}
		}
	}

	cmd := exec.Command("jekyll", "build", "--no-watch", "--quiet", "--safe", "-s", repoPath, "-d", sitePath)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
	}

	return dest.SetProto(&resp)
}
