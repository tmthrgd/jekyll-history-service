FROM envygeeks/alpine
MAINTAINER Tom Thorogood <me@tomthorogood.co.uk>

ENV \
	LANGUAGE=en_US \
	LANG=en_US.UTF-8 \
	LC_ALL=en_US \

	JEKYLL_ENV=production

RUN \
	apk add --update zlib-dev build-base libxml2-dev libxslt-dev readline-dev \
		libffi-dev ruby-dev yaml-dev zlib-dev libxslt-dev readline-dev libxml2-dev \
		libffi-dev ruby-dev yaml-dev zlib libxml2 build-base ruby-io-console readline \
		libxslt ruby yaml libffi nodejs ruby-irb ruby-json ruby-rake ruby-rdoc \
		git

RUN \
	yes | gem update --no-document -- --use-system-libraries && \
	yes | gem update --system --no-document -- --use-system-libraries

RUN \
	yes | gem install bundler --no-document && \
	docker-helper configure_nokogiri

RUN apk add --update py-pygments

RUN yes | docker-helper ruby-install-gem "jekyll@3.1.6"
RUN yes | docker-helper ruby-install-gem pygments.rb
RUN yes | docker-helper ruby-install-gem RedCloth
RUN yes | docker-helper ruby-install-gem kramdown
RUN \
	yes | docker-helper ruby-install-gem jemoji && \
	yes | docker-helper ruby-install-gem jekyll-assets && \
	yes | docker-helper ruby-install-gem jekyll-sitemap && \
	yes | docker-helper ruby-install-gem jekyll-mentions && \
	yes | docker-helper ruby-install-gem jekyll-coffeescript && \
	yes | docker-helper ruby-install-gem jekyll-sass-converter && \
	yes | docker-helper ruby-install-gem jekyll-redirect-from && \
	yes | docker-helper ruby-install-gem jekyll-paginate && \
	yes | docker-helper ruby-install-gem jekyll-compose && \
	yes | docker-helper ruby-install-gem jekyll-feed
RUN yes | docker-helper ruby-install-gem rdiscount

RUN \
	docker-helper add-user-1000 jekyll && \
	echo 'jekyll ALL=NOPASSWD:ALL' >> /etc/sudoers

RUN ln -s $(ruby -e 'print Gem.bin_path("jekyll", "jekyll")') /usr/local/bin/jekyll.orig

RUN gem clean

RUN apk del zlib-dev build-base libxml2-dev libxslt-dev readline-dev libffi-dev \
	ruby-dev yaml-dev zlib-dev libxslt-dev readline-dev libxml2-dev \
	libffi-dev ruby-dev yaml-dev zlib libxml2 build-base

RUN docker-helper cleanup

ADD jekyll /usr/local/bin
