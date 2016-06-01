// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
	"github.com/mitchellh/goamz/s3"
)

const sniffLen = 512

type buildJekyllGetter struct {
	WorkingDirectory string

	ExecuteJekyll func(src, dst string) error

	S3Bucket *s3.Bucket

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

	tagPath := filepath.Join(tag[0:1], tag[1:2], tag[2:])

	basePath := filepath.Join(bj.WorkingDirectory, tagPath)

	if !debug {
		defer os.RemoveAll(basePath)
	}

	repoPath := filepath.Join(basePath, "repo")
	sitePath := filepath.Join(basePath, "site")

	if list, err := bj.S3Bucket.List(tagPath, "/", "", 1); err == nil && len(list.CommonPrefixes) != 0 {
		return dest.SetProto(&resp)
	} else if err != nil {
		log.Printf("%[1]T: %[1]v", err)
	}

	u, gresp, err := bj.GithubClient.Repositories.GetArchiveLink(user, repo, github.Tarball, &github.RepositoryContentGetOptions{
		Ref: commit,
	})
	if err != nil {
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
		resp.Code = http.StatusBadGateway
		return dest.SetProto(&resp)
	}

	if verbose {
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

		_, err = copyBuffer(file, tarReader)
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

	executeJekyll := bj.ExecuteJekyll
	if executeJekyll == nil {
		executeJekyll = defaultExecuteJekyll
	}

	if err := executeJekyll(repoPath, sitePath); err != nil {
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
		return dest.SetProto(&resp)
	}

	if err := filepath.Walk(sitePath, func(filePath string, info os.FileInfo, err error) error {
		if info == nil {
			return &os.PathError{Op: "open", Path: filePath, Err: errors.New("failed to get file info")}
		}

		if info.IsDir() {
			return nil
		}

		if info.Mode()&(os.ModeDir|os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
			return &os.PathError{Op: "open", Path: filePath, Err: errors.New("not a regular file")}
		}

		f, err := os.Open(filePath)
		if err != nil {
			return err
		}

		ctype := mime.TypeByExtension(filepath.Ext(filePath))
		if len(ctype) == 0 {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(f, buf[:])

			ctype = http.DetectContentType(buf[:n])

			if _, err := f.Seek(0, os.SEEK_SET); err != nil {
				f.Close()
				return err
			}
		}

		size := info.Size()

		var r io.Reader = f
		var encoding []string

		if size > 1024 {
			buf := bufferPool.Get().(*bytes.Buffer)
			defer bufferPool.Put(buf)
			buf.Reset()

			gzw := gzip.NewWriter(buf)

			if _, err := copyBuffer(gzw, f); err != nil {
				return err
			}

			if err := gzw.Close(); err != nil {
				return err
			}

			if bufLen := int64(buf.Len()); bufLen < size {
				r = buf
				size = bufLen
				encoding = []string{"gzip"}
			} else if _, err := f.Seek(0, os.SEEK_SET); err != nil {
				return err
			}
		}

		err = bj.S3Bucket.PutReaderHeader(filepath.Join(tagPath, filePath[len(repoPath):]), r, size, map[string][]string{
			"Cache-Control":       {builtRepoCacheControl},
			"Content-Encoding":    encoding,
			"Content-Type":        {ctype},
			"x-amz-storage-class": {"REDUCED_REDUNDANCY"},
		}, "")
		f.Close()
		return err
	}); err != nil {
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
	}

	return dest.SetProto(&resp)
}
