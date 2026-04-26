FROM golang:1.23-alpine

RUN apk add --no-cache python3 protobuf

WORKDIR /workspace
COPY . .

CMD ["sh", "-c", "cd events && go test ./... && cd ../hot-state && go test ./... && cd /workspace && python3 -m unittest discover quantum-sim/tests && protoc -I proto --descriptor_set_out=/tmp/rapid_saas_shared_v1.pb proto/rapid_saas/shared/v1/*.proto"]
