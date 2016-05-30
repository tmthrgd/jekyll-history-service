// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import "fmt"

var (
	version string

	fullVersionStr string
)

func init() {
	if len(version) != 0 {
		fullVersionStr = fmt.Sprintf("jekyll-history-service (%s)", version)
	} else {
		fullVersionStr = "jekyll-history-service"
	}
}
