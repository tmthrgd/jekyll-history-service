// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/groupcache"
)

type notFoundError []byte

func (notFoundError) Error() string {
	return os.ErrNotExist.Error()
}

type builtFileGetter struct {
	SiteBasePath string
}

func (bf builtFileGetter) Get(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	parts := strings.Split(key, "\x00")
	if len(parts) != 2 {
		return &httpError{errors.New("invalid key"), http.StatusBadRequest}
	}

	tag, file := parts[0], parts[1]

	dir := http.Dir(filepath.Join(bf.SiteBasePath, tag[0:1], tag[1:2], tag[2:]))

	f, err := dir.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return bf.TryFind404(ctx, tag, dir, dest)
		}

		return err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	if stat.Mode()&(os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
		f.Close()
		return errors.New("not a regular file")
	}

	if stat.IsDir() {
		f.Close()

		if f, err = dir.Open(strings.TrimSuffix(file, "/") + "/index.html"); err != nil {
			if os.IsNotExist(err) {
				return bf.TryFind404(ctx, tag, dir, dest)
			}

			return err
		}
	}

	b, err := ioutil.ReadAll(f)

	f.Close()

	if err != nil {
		return err
	}

	return dest.SetBytes(b)
}

func (bf builtFileGetter) TryFind404(_ groupcache.Context, tag string, dir http.Dir, dest groupcache.Sink) error {
	f, err := dir.Open("/404.html")
	if err != nil {
		return err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	if stat.Mode()&(os.ModeDir|os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
		f.Close()
		return errors.New("not a regular file")
	}

	b, err := ioutil.ReadAll(f)

	f.Close()

	if err != nil {
		return err
	}

	return notFoundError(b)
}

var builtFiles *groupcache.Group
