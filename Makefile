all:
	go build -gcflags '-N -l' -buildmode=c-shared -o server/libsrx_grpc_server.so  server/srx_grpc_server.go
	go build -gcflags '-N -l' -buildmode=c-shared -o client/libsrx_grpc_client.so  client/srx_grpc_client.go


proto:
	protoc -I=. --go_out=plugins=grpc:. ./srx_grpc_v6.proto
