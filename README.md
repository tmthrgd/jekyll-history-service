# jekyll-history-service

jekyll-history-service is a hosted service that downloads a git repository and runs `jekyll build` at
any commit in its history.

## Download:

```
go get github.com/tmthrgd/jekyll-history-service && go install -ldflags "-X 'main.version=$(git rev-parse --short HEAD)$(git diff-files --quiet || echo -n -dirty)'" github.com/tmthrgd/jekyll-history-service
```

## Run:

`jekyll-history-service`

## License

Unless otherwise noted, the jekyll-history-service source files are distributed under the Modified BSD
License found in the LICENSE file.
