# jekyll-history-service

jekyll-history-service is a hosted service that downloads a git repository and runs `jekyll build` at
any commit in its history.

## Download:

	go get github.com/tmthrgd/jekyll-history-service && go install github.com/tmthrgd/jekyll-history-service

## Environment Variables:

The following environment variables **must** be set before use:

	AWS_ACCESS_KEY_ID=
	AWS_SECRET_ACCESS_KEY=
	
	S3_BUCKET=

The following environment variables **should** be set before use:

	GITHUB_CLIENT_ID=
	GITHUB_CLIENT_SECRET=
	
	S3_ENDPOINT=

If not set, `S3_ENDPOINT` defaults to `us-east-1`.

If a `.env` file exists in the working directory, environment variables will be read from it.

## Run:

	jekyll-history-service

## License

Unless otherwise noted, the jekyll-history-service source files are distributed under the Modified BSD
License found in the LICENSE file.
