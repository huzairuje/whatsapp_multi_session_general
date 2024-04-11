####################################
# STEP 1 build executable binary
####################################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
# gcc is required for cgo.
RUN apk update && apk add --no-cache git gcc libc-dev
RUN apk add ca-certificates
WORKDIR $GOPATH/src/whatsapp_multi_session_general

#copy all the content to container
COPY . .

##Fix go mod cant download without using proxy
ENV GOPROXY="https://goproxy.cn,direct"

# Build the binary
RUN export CGO_ENABLED=1 && go build -o /go/bin/whatsapp_multi_session_general

#move the config
COPY config.local.yaml /go/bin/config.local.yaml

#change the permission on binary
RUN chmod +x /go/bin/whatsapp_multi_session_general

##############################################
# STEP 2 build a small image using scratch
##############################################
FROM alpine

EXPOSE 7373

# Copy our static executable.
COPY --from=builder /go/bin/whatsapp_multi_session_general /whatsapp_multi_session_general
COPY --from=builder /go/bin/config.local.yaml /config.local.yaml

# Run the entrypoints.
ENTRYPOINT [ "./whatsapp_multi_session_general" ]