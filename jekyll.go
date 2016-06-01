// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"encoding/json"
	"os"
	"os/exec"
)

var defaultExecuteJekyll func(src, dst string) error

func init() {
	var err error
	if defaultExecuteJekyll, err = getExecuteShellJekyll(""); err != nil {
		panic(err)
	}
}

func getExecuteShellJekyll(optsflag string) (func(src, dst string) error, error) {
	opts := struct {
		Env  []string
		Args []string
	}{
		Env:  []string{},
		Args: []string{"--safe"},
	}

	if len(optsflag) != 0 {
		if err := json.Unmarshal([]byte(optsflag), &opts); err != nil {
			return nil, err
		}
	}

	args := []string{"--no-watch"}

	if debug {
		args = append(args, "--trace", "--verbose")
	}

	if !verbose {
		args = append(args, "--quiet")
	}

	args = append(args, opts.Args...)

	return func(src, dst string) error {
		cmd := exec.Command("jekyll", append([]string{"build", "-s", src, "-d", dst}, args...)...)
		cmd.Dir = src
		cmd.Env = opts.Env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}, nil
}
