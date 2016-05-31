// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"os"
	"os/exec"
)

func executeShellJekyll(src, dst string) error {
	cmd := exec.Command("jekyll", "build", "--no-watch", "--quiet", "--safe", "-s", src, "-d", dst)
	cmd.Dir = src
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
