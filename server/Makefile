all:
	go build -gcflags '-N -l' srx_grpc_server.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_server.so  srx_grpc_server.go
	gcc -g -O0 -o CImple_srx_grpc_server CImple_srx_grpc_server.c ./libsrx_grpc_server.so \
	-I${GOPATH}/src/srx_grpc_v6/srx-server/../_inst/include/srx \
	-I${GOPATH}/src/srx_grpc_v6/srx-server/src/ \
	-I${GOPATH}/src/srx_grpc_v6/srx-server/../_inst/include \
	-I${GOPATH}/src/srx_grpc_v6/srx-server/src/../extras/local/include \
	-Wl,--unresolved-symbols=ignore-all

go-old:
	go build -gcflags '-N -l' srx_grpc_server.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_server.so  srx_grpc_server.go

prefix = /opt/project/gobgp_test/gowork/src/srx_grpc_v6/_inst
ROOT= /opt/project/gobgp_test/gowork/src/srx_grpc_v6
SUBMODULE=
SRX_SERVER = srx-server
SRC = src
SRX_EXTRAS = $(SRX_SERVER)/extras



CGO_CFLAGS_SERVER="-DUSE_GRPC -g -Wall -I$(prefix)/include/ -I$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC) -I$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC)/client -I$(ROOT)/$(SUBMODULE)/$(SRX_EXTRAS)/local/include"

CGO_LDFLAGS_SERVER="-L$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC)/.libs -lgrpc_service -Wl,-rpath -Wl,$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC)/.libs -Wl,--unresolved-symbols=ignore-all"



go:
	@echo go module compile and build
	CGO_CFLAGS=$(CGO_CFLAGS_SERVER) CGO_LDFLAGS=$(CGO_LDFLAGS_SERVER) go build -gcflags '-N -l' ./srx_grpc_server.go
	CGO_CFLAGS=$(CGO_CFLAGS_SERVER) CGO_LDFLAGS=$(CGO_LDFLAGS_SERVER) go build -gcflags '-N -l' -buildmode=c-shared \
			   -o ./libsrx_grpc_server.so  ./srx_grpc_server.go



stream:
	go build -gcflags '-N -l' srx_grpc_server_stream.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_server_stream.so  srx_grpc_server_stream.go
	gcc -g -O0 -o CImple_srx_grpc_server_stream CImple_srx_grpc_server.c ./libsrx_grpc_server_stream.so \
	-I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/srx_install/include/srx \
	-I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/ \
	-I/opt/project/srx_test1/srx/../_inst//include \
	-I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/../extras/local/include \
	-Wl,--unresolved-symbols=ignore-all

callback:
	go build -gcflags '-N -l' srx_grpc_server_callback.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_server_callback.so  srx_grpc_server_callback.go
	gcc -g -O0 -o CImple_srx_grpc_server_callback CImple_srx_grpc_server_callback.c ./libsrx_grpc_server_callback.so \
	-I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/srx_install/include/srx \
	-I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/ \
	-I/opt/project/srx_test1/srx/../_inst//include \
	-I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/../extras/local/include \
	-Wl,--unresolved-symbols=ignore-all





protoc:
	@echo "Generating Go protobuf files"
	cd ../
	protoc -I=. --go_out=plugins=grpc:. ./srx_grpc.proto


server: protoc
	@echo "Building Server"
	go build -gcflags '-N -l' srx_grpc_server.go


.PHONY: all server client stream callback
