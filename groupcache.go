// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"hash/crc32"

	"github.com/golang/groupcache"
)

func getGroupcache(getter *buildJekyllGetter) (*groupcache.Group, *groupcache.HTTPPool, *groupcache.HTTPPoolOptions) {
	buildJekyll := groupcache.NewGroup("build-jekyll", 1<<20, getter)

	castagnoli := crc32.MakeTable(crc32.Castagnoli)

	poolOpts := &groupcache.HTTPPoolOptions{
		BasePath: "/_groupcache/",

		HashFn: func(data []byte) uint32 {
			if idx := bytes.IndexByte(data, 0x00); idx != -1 {
				return crc32.Checksum(data[:idx], castagnoli)
			}

			return crc32.Checksum(data, castagnoli)
		},
	}
	httpPool := groupcache.NewHTTPPoolOpts("http://jekyllhistory.org:8080", poolOpts)

	return buildJekyll, httpPool, poolOpts
}
