FROM golang:1.20-buster as builder

COPY . /code/cloud-provider-ionoscloud
WORKDIR /code/cloud-provider-ionoscloud
RUN CGO_ENABLED=0 go build

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /code/cloud-provider-ionoscloud/cloud-provider-ionoscloud /usr/bin/cloud-provider-ionoscloud

WORKDIR /opt

# replace with your desire device count
CMD ["/usr/bin/cloud-provider-ionoscloud"]
