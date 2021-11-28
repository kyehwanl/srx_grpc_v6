
.PHONY: all go go-old

CGO_CFLAGS_CLIENT_OLD ="-g -Wall -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/../_inst/include/ -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/ -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/../extras/local/include"

CGO_LDFLAGS_CLIENT_OLD = "-L/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/.libs -lgrpc_client_service -Wl,-rpath -Wl,/opt/project/gobgp_test/gowork/src/srx_grpc_v6//srx-server//src/.libs -Wl,--unresolved-symbols=ignore-all"


prefix = /opt/project/gobgp_test/gowork/src/srx_grpc_v6/_inst
ROOT= /opt/project/gobgp_test/gowork/src/srx_grpc_v6
SUBMODULE=
SRX_SERVER = srx-server
SRC = src
SRX_EXTRAS = $(SRX_SERVER)/extras



CGO_CFLAGS_CLIENT="-g -Wall -I$(prefix)/include/ -I$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC) -I$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC)/client -I$(ROOT)/$(SUBMODULE)/$(SRX_EXTRAS)/local/include"

CGO_LDFLAGS_CLIENT="-L$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC)/.libs -lgrpc_client_service -Wl,-rpath -Wl,$(ROOT)/$(SUBMODULE)/$(SRX_SERVER)/$(SRC)/.libs -Wl,--unresolved-symbols=ignore-all"



all:
	go build -gcflags '-N -l' srx_grpc_client.go
	go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_client.so  srx_grpc_client.go
	gcc -g -O0 -o CImple_srx_grpc_client CImple_srx_grpc_client.c ./libsrx_grpc_client.so \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/../_inst/include/srx \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/src/ \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/../_inst/include \
		-I${GOPATH}/src/srx_grpc_v6/srx-server/src/../extras/local/include \
		-Wl,--unresolved-symbols=ignore-all -lpthread

go-old:
	CGO_CFLAGS=$(CGO_CFLAGS_CLIENT_OLD) CGO_LDFLAGS=$(CGO_LDFLAGS_CLIENT_OLD) \
			   go build -gcflags '-N -l' srx_grpc_client.go
	CGO_CFLAGS=$(CGO_CFLAGS_CLIENT_OLD) CGO_LDFLAGS=$(CGO_LDFLAGS_CLIENT_OLD) \
			   go build -gcflags '-N -l' -buildmode=c-shared -o libsrx_grpc_client.so  srx_grpc_client.go



go: 
	CGO_CFLAGS=$(CGO_CFLAGS_CLIENT) CGO_LDFLAGS=$(CGO_LDFLAGS_CLIENT) go build -gcflags '-N -l' srx_grpc_client.go
	CGO_CFLAGS=$(CGO_CFLAGS_CLIENT) CGO_LDFLAGS=$(CGO_LDFLAGS_CLIENT) go build -gcflags '-N -l' -buildmode=c-shared \
			   -o ./libsrx_grpc_client.so  srx_grpc_client.go
