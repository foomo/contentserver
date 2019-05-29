##############################
###### STAGE: BUILD     ######
##############################
FROM golang:1.12.5 AS build-env

WORKDIR /src

COPY ./go.mod ./go.sum ./
RUN go mod download && go mod vendor && go install -i ./vendor/...

# Import the code from the context.
COPY ./ ./

RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0  go build -o /contentserver

##############################
###### STAGE: PACKAGE   ######
##############################
FROM alpine

ENV CONTENT_SERVER_ADDR=0.0.0.0:80
ENV CONTENT_SERVER_VAR_DIR=/var/lib/contentserver
ENV LOG_JSON=1

RUN apk add --update --no-cache ca-certificates curl bash && rm -rf /var/cache/apk/*

COPY --from=build-env /contentserver /usr/sbin/contentserver


VOLUME $CONTENT_SERVER_VAR_DIR

ENTRYPOINT ["/usr/sbin/contentserver"]

CMD ["-address=$CONTENT_SERVER_ADDR", "-var-dir=$CONTENT_SERVER_VAR_DIR"]

EXPOSE 80
EXPOSE 9200
