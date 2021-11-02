all:
	go build -gcflags '-N -l' srx_grpc_client.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_client.so  srx_grpc_client.go
	gcc -g -O0 -o CImple_srx_grpc_client CImple_srx_grpc_client.c ./libsrx_grpc_client.so \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/../_inst/include/srx \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/src/ \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/../_inst/include \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/src/../extras/local/include \
		-Wl,--unresolved-symbols=ignore-all -lpthread

go: 
	go build -gcflags '-N -l' srx_grpc_client.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_client.so  srx_grpc_client.go
