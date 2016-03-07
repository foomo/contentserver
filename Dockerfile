FROM scratch

COPY bin/contentserver /usr/sbin/contentserver

# install ca root certificates
# https://curl.haxx.se/docs/caextract.html
# http://blog.codeship.com/building-minimal-docker-containers-for-go-applications/
ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificates.crt

ENV CONTENT_SERVER_LOG_LEVEL=error
ENV CONTENT_SERVER_ADDR=0.0.0.0:80
ENV CONTENT_SERVER_PROTOCOL=tcp
ENV CONTENT_SERVER_VAR_DIR=/var/lib/contentserver

VOLUME $CONTENT_SERVER_VAR_DIR
EXPOSE 80

ENTRYPOINT ["/usr/sbin/contentserver"]

CMD ["-address=$CONTENT_SERVER_ADDR", "-logLevel=$CONTENT_SERVER_LOG_LEVEL", "-protocol=$CONTENT_SERVER_PROTOCOL", "-vardir=$CONTENT_SERVER_VAR_DIR"]
