// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/goamz/s3"
)

var (
	builtRepoCacheControl = fmt.Sprintf("public, max-age=%d", (10*365*24*time.Hour)/time.Second)

	timeZero time.Time

	hostRegex = regexp.MustCompile(`^([0-9a-fA-F]{32}).jekyllhistory.org$`)
)

type repoSwitch struct {
	S3Bucket *s3.Bucket
}

func (rs repoSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	m := hostRegex.FindStringSubmatch(host)
	if m == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	tag := strings.ToLower(m[1])

	if strings.HasSuffix(r.URL.Path, "/index.html") {
		localRedirect(w, r, "./")
		return
	}

	h := w.Header()
	h.Set("Cache-Control", builtRepoCacheControl)
	h.Set("Etag", `"`+tag+`"`)

	if checkETag(w, r) {
		return
	}

	if filepath.Separator != '/' && strings.IndexRune(r.URL.Path, filepath.Separator) >= 0 || strings.Contains(r.URL.Path, "\x00") {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	name := r.URL.Path

	if strings.HasSuffix(name, "/") {
		name += "/index.html"
	}

	basePath := filepath.Join(tag[0:1], tag[1:2], tag[2:])
	fullPath := filepath.Join(basePath, filepath.FromSlash(path.Clean("/"+name)))

	switch r.Method {
	case http.MethodGet:
		resp, err := rs.S3Bucket.GetResponse(fullPath)

		if err == nil {
			for _, k := range [...]string{"Content-Length", "Content-Type"} {
				for _, v := range resp.Header[k] {
					h.Add(k, v)
				}
			}

			if modtime, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified")); err == nil && checkLastModified(w, r, modtime, 0) {
				resp.Body.Close()
				return
			}

			if err = rs.serveS3Response(w, r, resp, resp.StatusCode); err != nil {
				log.Printf("%[1]T: %[1]v", err)

				h.Del("Etag")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			return
		}

		s3err, ok := err.(*s3.Error)
		if !ok {
			log.Printf("%[1]T: %[1]v", err)

			h.Del("Etag")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if s3err.StatusCode != 404 {
			log.Printf("%[1]T: %[1]v", err)

			h.Del("Etag")
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		resp, err = rs.S3Bucket.GetResponse(filepath.Join(basePath, "/404.html"))
		if err != nil {
			if s3err, ok := err.(*s3.Error); ok && s3err.StatusCode != 404 {
				log.Printf("%[1]T: %[1]v", err)

				h.Del("Etag")
			}

			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		for _, k := range [...]string{"Content-Length", "Content-Type"} {
			for _, v := range resp.Header[k] {
				h.Add(k, v)
			}
		}

		if err = rs.serveS3Response(w, r, resp, http.StatusNotFound); err != nil {
			log.Printf("%[1]T: %[1]v", err)

			h.Del("Etag")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	case http.MethodHead:
		resp, err := rs.S3Bucket.Head(fullPath)

		if err == nil {
			for _, k := range [...]string{"Content-Encoding", "Content-Length", "Content-Type"} {
				for _, v := range resp.Header[k] {
					h.Add(k, v)
				}
			}

			if modtime, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified")); err == nil && checkLastModified(w, r, modtime, 0) {
				return
			}

			w.WriteHeader(resp.StatusCode)
			return
		}

		if s3err, ok := err.(*s3.Error); ok {
			if s3err.StatusCode == 404 {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			} else {
				log.Printf("%[1]T: %[1]v", err)
				http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			}
		} else {
			log.Printf("%[1]T: %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead+", "+http.MethodOptions)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (repoSwitch) serveS3Response(w http.ResponseWriter, r *http.Request, resp *http.Response, code int) error {
	if encoding := strings.TrimSpace(resp.Header.Get("Content-Encoding")); strings.ToLower(encoding) == "gzip" {
		h := w.Header()
		h.Set("Vary", "Accept-Encoding")

		canGzip := false

		if accept := r.Header.Get("Accept-Encoding"); len(accept) != 0 {
			for _, v := range strings.Split(accept, ",") {
				if idx := strings.Index(v, ";"); idx != -1 {
					v = v[:idx]
				}

				if strings.ToLower(strings.TrimSpace(v)) == "gzip" {
					canGzip = true
					break
				}
			}
		}

		if canGzip {
			h.Set("Content-Encoding", "gzip")
		} else {
			h.Del("Content-Length")

			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}

			w.WriteHeader(code)

			io.Copy(w, gr)
			gr.Close()
			resp.Body.Close()
			return nil
		}
	} else if len(encoding) != 0 {
		// unkown encoding
		w.Header().Set("Content-Encoding", encoding)
	}

	w.WriteHeader(code)

	io.Copy(w, resp.Body)
	resp.Body.Close()
	return nil
}
