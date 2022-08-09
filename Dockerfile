FROM alpine:3.16 AS build

ARG CAPROVER_GIT_COMMIT_SHA=""
VOLUME ["/data"]
COPY . /oko-server/git

RUN apk update && \
    apk add go && \
    cd /oko-server/git && \
    go build -ldflags "-X \"main.sha1ver=${CAPROVER_GIT_COMMIT_SHA:-$(cat .git/$(cat .git/HEAD | sed 's|ref: ||g'))}\" -X \"main.buildTime=$(date -Iseconds)\""

FROM alpine:3.16
WORKDIR /oko-server
VOLUME [ "/data" ]

COPY --from=build /oko-server/git/oko-server/ /oko-server/

ENTRYPOINT ["/oko-server/oko-server"]

