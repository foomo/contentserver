FROM alpine:3.21.3

RUN apk --no-cache add ca-certificates

RUN addgroup --system --gid 1001 contentserver
RUN adduser --system --uid 1001 contentserver

COPY contentserver /usr/bin/

RUN mkdir "/var/lib/contentserver" && \
		chmod 0700 "/var/lib/contentserver" && \
    chown contentserver:contentserver "/var/lib/contentserver"

USER contentserver

ENTRYPOINT ["contentserver"]
