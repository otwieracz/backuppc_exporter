FROM golang:1.12 as build

# golang deps
WORKDIR /tmp/app/
COPY ./src /tmp/src/

WORKDIR /go/src/backuppc_exporter/src
COPY ./main.go /go/src/backuppc_exporter/src
RUN mkdir /app/ \
    && cp -a /tmp/src/entrypoint.sh /app/ && chmod 555 /app/entrypoint.sh \
    && go get -u github.com/prometheus/client_golang/prometheus \
    && go build -o /app/backuppc_exporter

#############################################
# FINAL IMAGE
#############################################
FROM alpine
RUN apk add --no-cache \
      libc6-compat \
    	ca-certificates \
    	wget \
    	curl
COPY --from=build /app/ /app/
USER 1000
ENTRYPOINT ["/app/entrypoint.sh"]
