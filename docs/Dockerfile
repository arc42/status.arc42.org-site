FROM ruby:3.2-slim

LABEL version="1.0.0"
LABEL description="status.arc42.org Jekyll dev image"
LABEL vendor="arc42 (Gernot Starke)"

RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential git libcurl4 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp/bundle

RUN gem install bundler:2.3.25

COPY Gemfile Gemfile.lock ./

RUN bundle _2.3.25_ config set path /usr/local/bundle \
    && bundle _2.3.25_ install --frozen --jobs 4 --retry 3 \
    && mkdir -p /opt/site-deps \
    && sha256sum Gemfile.lock | awk '{ print $1 }' > /opt/site-deps/.gemfile-lock.sha256

COPY docker/jekyll-entrypoint.sh /usr/local/bin/jekyll-entrypoint
RUN chmod +x /usr/local/bin/jekyll-entrypoint

WORKDIR /site

EXPOSE 4270

ENTRYPOINT ["jekyll-entrypoint"]
CMD ["bundle", "exec", "jekyll", "serve", "--host", "0.0.0.0", "--port", "4270", "--watch"]
