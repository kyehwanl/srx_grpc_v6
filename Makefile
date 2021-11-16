all:
	go build -gcflags '-N -l' -buildmode=c-shared -o server/libsrx_grpc_server.so  server/srx_grpc_server.go
	go build -gcflags '-N -l' -buildmode=c-shared -o client/libsrx_grpc_client.so  client/srx_grpc_client.go


proto:
	protoc -I=. --go_out=plugins=grpc:. ./srx_grpc_v6.proto


# this service will be used in Makefile at srx-server/src
service:
	/usr/bin/libtool --tag=CC   --mode=compile gcc -DHAVE_CONFIG_H  -c -o grpc_service.lo server/grpc_service.c \
    	-I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src \
    	-I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/../extras/local/include \
    	-I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/../_inst/include
	/usr/bin/libtool --tag=CC   --mode=link   gcc -DHAVE_CONFIG_H  -o libgrpc_service.la grpc_service.lo -rpath `pwd`/.libs \
    	-L/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/.libs/ -lsrx_util
	/usr/bin/libtool --tag=CC   --mode=install cp libgrpc_service.la ${GOPATH}/src/srx_grpc_v6/_inst/lib64
	/usr/bin/libtool --tag=CC   --finish ${GOPATH}/src/srx_grpc_v6/_inst/lib64

