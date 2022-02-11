FROM alpine:3.15.0

RUN apk add --no-cache git go && \
    mkdir /oko-server && \
    cd /oko-server && \
    git clone https://github.com/Cernobor/oko-server.git /oko-server/git && \
    cd /oko-server/git && \
    go build && \
    cp /oko-server/git/oko-server /oko-server/ && \
    cd /oko-server && \
    rm -rf /oko-server/git && \
    mkdir /data && \
    apk del git go && \
    rm -rf /root/go && \
    echo -e '#!/bin/sh\n/oko-server/oko-server "$@"' > /oko-server/entrypoint.sh && \
    chmod +x /oko-server/entrypoint.sh
VOLUME ["/data"]

ENTRYPOINT ["/oko-server/entrypoint.sh"]