FROM alpine:3.19.1

RUN apk --no-cache add ca-certificates
RUN addgroup -S contentserver && \
    adduser -S -g contentserver contentserver

COPY contentserver /usr/bin/

RUN mkdir "/var/lib/contentserver" && \
		chmod 0700 "/var/lib/contentserver" && \
    chown contentserver:contentserver "/var/lib/contentserver"

USER contentserver
ENTRYPOINT ["contentserver"]
