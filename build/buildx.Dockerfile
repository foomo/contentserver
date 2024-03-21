# syntax=docker/dockerfile:1.4
FROM golang:1.21

# related to https://github.com/golangci/golangci-lint/issues/3107
ENV GOROOT /usr/local/go

# Allow to download a more recent version of Go.
# https://go.dev/doc/toolchain
# GOTOOLCHAIN=auto is shorthand for GOTOOLCHAIN=local+auto
ENV GOTOOLCHAIN auto

# Set all directories as safe
RUN git config --global --add safe.directory '*'

COPY contentserver /usr/bin/
ENTRYPOINT ["/usr/bin/contentserver"]

ENV CONTENT_SERVER_ADDRESS=0.0.0.0:8080
ENV CONTENT_SERVER_VAR_DIR=/var/lib/contentserver
ENV LOG_JSON=1

EXPOSE 8080
EXPOSE 9200

CMD ["-address=$CONTENT_SERVER_ADDRESS", "-var-dir=$CONTENT_SERVER_VAR_DIR"]
