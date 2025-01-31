FROM golang:1.23.5

RUN apt-get update; apt-get install --yes unzip

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# TODO: Compile ourselves?
RUN mkdir -p /opt/protoc
RUN wget -N -O /opt/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.12.4/protoc-3.12.4-linux-x86_64.zip

RUN cd /opt; unzip protoc.zip; rm protoc.zip

ENTRYPOINT [ "/opt/bin/protoc" ]
