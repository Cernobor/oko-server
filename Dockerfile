FROM alpine:3.15.0 AS build

VOLUME ["/data"]
COPY . /oko-server/git

RUN apk add --no-cache go && \
    cd /oko-server/git && \
    go build

FROM alpine:3.15.0
WORKDIR /oko-server
VOLUME [ "/data" ]

RUN echo -e '#!/bin/sh\n/oko-server/oko-server "$@"' > /oko-server/entrypoint.sh && \
    chmod +x /oko-server/entrypoint.sh
COPY --from=build /oko-server/git/oko-server/ /oko-server/

ENTRYPOINT ["/oko-server/entrypoint.sh"]

