// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/groupcache"
)

type builtFileGetter struct {
	SiteBasePath string
}

func (bf builtFileGetter) Get(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	var resp BuiltFileResponse

	parts := strings.Split(key, "\x00")
	if len(parts) != 2 {
		resp.Error = "invalid key"
		resp.Code = http.StatusBadRequest
		return dest.SetProto(&resp)
	}

	tag, file := parts[0], parts[1]

	dir := http.Dir(filepath.Join(bf.SiteBasePath, tag[0:1], tag[1:2], tag[2:]))

	f, err := dir.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return bf.TryFind404(ctx, tag, dir, dest)
		}

		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
		return dest.SetProto(&resp)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()

		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
		return dest.SetProto(&resp)
	}

	if stat.Mode()&(os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
		f.Close()

		resp.Error = "not a regular file"
		resp.Code = http.StatusForbidden
		return dest.SetProto(&resp)
	}

	if stat.IsDir() {
		f.Close()

		if f, err = dir.Open(strings.TrimSuffix(file, "/") + "/index.html"); err != nil {
			if os.IsNotExist(err) {
				return bf.TryFind404(ctx, tag, dir, dest)
			}

			resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
			return dest.SetProto(&resp)
		}
	}

	if b, err := ioutil.ReadAll(f); err != nil {
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
	} else {
		resp.Data = b
		resp.ModTime = stat.ModTime().Unix()
	}

	f.Close()

	return dest.SetProto(&resp)
}

func (bf builtFileGetter) TryFind404(_ groupcache.Context, tag string, dir http.Dir, dest groupcache.Sink) error {
	var resp BuiltFileResponse

	f, err := dir.Open("/404.html")
	if err != nil {
		if os.IsNotExist(err) {
			resp.Error = "not found"
			resp.Code = http.StatusNotFound
		} else {
			resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
		}

		return dest.SetProto(&resp)
	}

	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
		return dest.SetProto(&resp)
	}

	if stat.Mode()&(os.ModeDir|os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
		resp.Error = "not a regular file"
		resp.Code = http.StatusForbidden
		return dest.SetProto(&resp)
	}

	if resp.Data, err = ioutil.ReadAll(f); err != nil {
		resp.Data = nil
		resp.Error = fmt.Sprintf("%[1]T: %[1]v", err)
	}

	resp.Code = http.StatusNotFound
	return dest.SetProto(&resp)
}
