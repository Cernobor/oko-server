FROM alpine:3.15.0

VOLUME ["/data"]
COPY . /oko-server/git

RUN apk add --no-cache go && \
    cd /oko-server/git && \
    go build && \
    cp /oko-server/git/oko-server /oko-server/ && \
    cd /oko-server && \
    rm -rf /oko-server/git && \
    apk del go && \
    rm -rf /root/go && \
    echo -e '#!/bin/sh\n/oko-server/oko-server "$@"' > /oko-server/entrypoint.sh && \
    chmod +x /oko-server/entrypoint.sh

ENTRYPOINT ["/oko-server/entrypoint.sh"]
