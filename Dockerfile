FROM golang:1.19-buster as builder

WORKDIR /app

RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  ca-certificates && \
  rm -rf /var/lib/apt/lists/*

ARG GOPROXY="https://goproxy.cn,direct"

COPY go.mod go.sum /app/
RUN go mod download
COPY . /app

RUN set -eux; \
  dpkgArch="$(dpkg --print-architecture | awk -F- '{ print $NF }')"; \
  CGO_ENABLED=0 GOOS=linux GOARCH=$dpkgArch go build \
  -a -installsuffix cgo \
  -o /bin/myapp .

FROM debian:bullseye

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8
ENV TZ "Asia/Shanghai"
ARG UID=1000
ARG GID=1000

RUN apt-get update && \
  apt-get -y install skopeo && \
  rm -rf /var/lib/apt/lists/*

RUN set -eux; \
  mkdir /app; \
  groupadd -r app --gid=${GID}; \
  useradd -r -g app --uid=${UID} --shell=/bin/bash app; \
  ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime; \
  echo ${TZ} > /etc/timezone; \
  chown -R app:app /app

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/index.html /app/
COPY --from=builder /bin/myapp /usr/local/bin/

ENV STATIC_ROOT /app
EXPOSE 8080

ENTRYPOINT ["myapp"]
