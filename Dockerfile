# Stage 1
FROM golang:1.15 as builder
RUN git clone https://github.com/joeydouglas/function-deploy-relay.git fdr
WORKDIR /go/fdr
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -o fdr

# Stage 2
FROM ubuntu:20.04
RUN apt-get update && apt-get install -y ca-certificates --no-install-recommends && rm -rf /var/lib/apt/lists/*
WORKDIR /root
COPY --from=builder /go/fdr .
CMD ["./fdr"]