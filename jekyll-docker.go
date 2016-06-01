// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/strslice"
	"github.com/docker/go-connections/tlsconfig"
	"golang.org/x/net/context"
)

func getExecuteDockerJekyll(optsflag string) (func(src, dst string) error, error) {
	opts := struct {
		Host string

		Image string

		Env  []string
		Args []string

		TLS struct {
			CACert string
			Cert   string
			Key    string

			Use    bool
			Verify bool
		}
	}{
		Host: client.DefaultDockerHost,

		Image: "jekyll/jekyll",

		TLS: struct {
			CACert string
			Cert   string
			Key    string

			Use    bool
			Verify bool
		}{
			CACert: "~/.docker/ca.pem",
			Cert:   "~/.docker/cert.pem",
			Key:    "~/.docker/key.pem",

			Verify: true,
		},
	}

	if len(optsflag) != 0 {
		if err := json.Unmarshal([]byte(optsflag), &opts); err != nil {
			return nil, err
		}
	}

	if len(opts.Host) == 0 || len(opts.Image) == 0 {
		return nil, fmt.Errorf("invalid options")
	}

	var httpClient *http.Client

	if opts.TLS.Use {
		tlsc, err := tlsconfig.Client(tlsconfig.Options{
			CAFile:             opts.TLS.CACert,
			CertFile:           opts.TLS.Cert,
			KeyFile:            opts.TLS.Key,
			InsecureSkipVerify: !opts.TLS.Verify,
		})
		if err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
		}
	}

	api, err := client.NewClient(opts.Host, "v1.23", httpClient, map[string]string{
		"User-Agent": fullVersionStr,
	})
	if err != nil {
		return nil, err
	}

	if _, _, err := api.ImageInspectWithRaw(context.Background(), opts.Image, false); err != nil {
		return nil, err
	}

	cmd := strslice.StrSlice{"jekyll", "build", "--no-watch", "-s", "/srv/src", "-d", "/srv/dst"}

	if debug {
		cmd = append(cmd, "--trace", "--verbose")
	}

	if !verbose {
		cmd = append(cmd, "--quiet")
	}

	cmd = append(cmd, opts.Args...)

	seenWarnings := make(map[string]struct{})
	var seenWarningsMu sync.Mutex

	return func(src, dst string) error {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}

		resp, err := api.ContainerCreate(context.Background(), &container.Config{
			AttachStdout: true,
			AttachStderr: true,

			Env: opts.Env,

			Cmd: cmd,

			Image: opts.Image,

			NetworkDisabled: true,
		}, &container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/srv/src:ro", src),
				fmt.Sprintf("%s:/srv/dst", dst),
			},

			NetworkMode: "none",

			AutoRemove: !debug,

			ReadonlyRootfs: true,

			Resources: container.Resources{
				Memory:     100 * 1024 * 1024,
				MemorySwap: 100 * 1024 * 1024,

				DiskQuota: 0,
			},
		}, nil, "")
		if err != nil {
			return err
		}

		if !debug {
			defer api.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{})
		}

		if len(resp.Warnings) != 0 {
			seenWarningsMu.Lock()

			var hadSeen int

			for _, warn := range resp.Warnings {
				if _, seen := seenWarnings[warn]; seen {
					hadSeen++
					continue
				}

				seenWarnings[warn] = struct{}{}

				log.Printf("warning from docker: %s", warn)
			}

			if hadSeen != 0 {
				log.Printf("saw %d already seen warnings", hadSeen)
			}

			seenWarningsMu.Unlock()
		}

		if err = api.ContainerStart(context.Background(), resp.ID, ""); err != nil {
			return err
		}

		if logs, err := api.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{
			ShowStdout: verbose,
			ShowStderr: true,

			Timestamps: true,
			Follow:     true,
		}); err != nil {
			log.Printf("%[1]T: %[1]v", err)
		} else {
			go func() {
				defer logs.Close()

				logHeader := bytes.NewReader([]byte(fmt.Sprintf("log from %s: ", resp.ID[:12])))

				var hdr [8]byte

				for {
					if _, err := io.ReadFull(logs, hdr[:]); err == io.EOF {
						break
					} else if err != nil {
						log.Printf("%[1]T: %[1]v", err)
						return
					}

					var out io.Writer

					switch hdr[0] {
					case 0: /* stdin */
						panic("unreachable")
					case 1: /* stdout */
						out = os.Stdout
					case 2: /* stderr */
						out = os.Stderr
					default:
						panic("unreachable")
					}

					size := binary.BigEndian.Uint32(hdr[4:])

					if _, err := io.Copy(out, io.MultiReader(logHeader, &io.LimitedReader{
						R: logs,
						N: int64(size),
					})); err != nil {
						log.Printf("%[1]T: %[1]v", err)
						return
					}

					if _, err := logHeader.Seek(0, 0); err != nil {
						log.Printf("%[1]T: %[1]v", err)
					}
				}
			}()
		}

		code, err := api.ContainerWait(context.Background(), resp.ID)
		if err != nil {
			return err
		}

		if code != 0 {
			return fmt.Errorf("exit status %d", code)
		}

		return nil
	}, nil
}
