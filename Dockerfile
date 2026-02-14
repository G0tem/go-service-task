# First stage - build
FROM golang:1.24.0 AS build

# `boilerplate` should be replaced with your project name
WORKDIR /go/src/server

# Copy all the Code and stuff to compile everything
COPY docs docs
COPY internal internal

COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go

ENV GO111MODULE=on
ENV GOPRIVATE=github.com/G0tem

# раскомментировать и прокинуть в compose переменную, если используем приватные пакеты (оставим для удобства)
# ARG GITHUB_TOKEN
# RUN if [ "x$GITHUB_TOKEN" == "x" ]; then \
#         mkdir /root/.ssh; \
#         ssh-keyscan github.com >> ~/.ssh/known_hosts; \
#         cp ./id_rsa /root/.ssh/id_rsa; \
#         git config --global url."git@github.com:".insteadOf "https://github.com/"; \
#     else \
#         git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; \
#     fi

# Downloads all the dependencies in advance
RUN go mod tidy

# Building the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o app ./main.go

# Final stage - bake release image
FROM alpine:3.19 AS release
WORKDIR /app
COPY --from=build /go/src/server/app /usr/bin/app
COPY --from=build /go/src/server/docs/ ./docs/

# Add Alpine mirrors and install packages with retry
RUN echo "https://mirror.yandex.ru/mirrors/alpine/v3.19/main" > /etc/apk/repositories && \
    echo "https://mirror.yandex.ru/mirrors/alpine/v3.19/community" >> /etc/apk/repositories && \
    for i in $(seq 1 3); do \
        apk update && \
        apk add --no-cache ca-certificates && \
        chmod +x /usr/bin/app && \
        break || sleep 1; \
    done

ENTRYPOINT ["/usr/bin/app"]