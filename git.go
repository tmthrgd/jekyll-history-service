// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"path/filepath"
	"os"
	"os/exec"

	"github.com/google/go-github/github"
)

func cloneGitRepo(repoURL, work string) (repoDir string, err error) {
	repoDir = filepath.Join(work, "repo")

	cmd := exec.Command("git", "clone", repoURL, repoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	return
}

func gitRepoCommits(repoDir string) ([]github.RepositoryCommit, error) {
	cmd = exec.Command("git", "log", "--oneline")
	cmd.Dir = repoDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var commits []github.RepositoryCommit

	scanner := bufio.NewScanner(&out)

	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), " ", 2)
		sha := line[0]
		message := line[1]

		commits = append(commits, github.RepositoryCommit{
			SHA: &sha,

			Commit: &github.Commit{
				Message: &message,
			},
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commits, nil
}
