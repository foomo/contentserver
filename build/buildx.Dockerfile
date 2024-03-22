FROM alpine:3.19.1

RUN apk --no-cache add ca-certificates
RUN addgroup -S contentserver && adduser -S -g contentserver contentserver

COPY contentserver /usr/bin/

USER contentserver
ENTRYPOINT ["contentserver"]
