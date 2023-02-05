FROM golang:1.19
WORKDIR /go/src
# make deps fetching cacheable
COPY go.mod go.sum ./
RUN go mod download
# build
COPY . .
RUN make build

# build real nftgw image
FROM gcr.io/distroless/static-debian11
COPY --from=0 --chown=root:root ./go/src/bin/nftgw /bin/nftgw
CMD ["/bin/nftgw"]