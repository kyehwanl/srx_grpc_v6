#!/bin/bash -x
LIBTOOL="/usr/bin/libtool --tag=CC"
INSTALL_DIR=${GOPATH}/src/srx_grpc/version6/_inst

cd ${GOPATH}/src/srx_grpc/version6/srx-server/src
$LIBTOOL   --mode=compile gcc -DHAVE_CONFIG_H  -c -o grpc_service.lo server/grpc_service.c \
    -I/opt/project/gobgp_test/gowork/src/srx_grpc/version6/srx-server/src \
    -I/opt/project/gobgp_test/gowork/src/srx_grpc/version6/srx-server/src/../extras/local/include \
    -I/opt/project/gobgp_test/gowork/src/srx_grpc/version6/srx-server/../_inst/include
$LIBTOOL   --mode=link   gcc -DHAVE_CONFIG_H  -o libgrpc_service.la grpc_service.lo #-rpath ${INSTALL_DIR}/lib64
$LIBTOOL   --tag=CC   --mode=install cp libgrpc_service.la ${INSTALL_DIR}/lib64
$LIBTOOL   --tag=CC   --finish ${INSTALL_DIR}/lib64

$LIBTOOL   --tag=CC   --mode=compile gcc -DHAVE_CONFIG_H  -c -o grpc_client_service.lo client/grpc_client_service.c \
    -I/opt/project/gobgp_test/gowork/src/srx_grpc/version6/srx-server/src \
    -I/opt/project/gobgp_test/gowork/src/srx_grpc/version6/srx-server/src/../extras/local/include \
    -I/opt/project/gobgp_test/gowork/src/srx_grpc/version6/srx-server/../_inst/include
$LIBTOOL   --mode=link   gcc -DHAVE_CONFIG_H  -o libgrpc_client_service.la grpc_client_service.lo #-rpath ${INSTALL_DIR}/lib64
$LIBTOOL   --tag=CC   --mode=install cp libgrpc_client_service.la ${INSTALL_DIR}/lib64
$LIBTOOL   --tag=CC   --finish ${INSTALL_DIR}/lib64

cd ${GOPATH}/src/srx_grpc/version6
go build -gcflags '-N -l' -buildmode=c-shared -o server/libsrx_grpc_server.so  server/srx_grpc_server.go
go build -gcflags '-N -l' -buildmode=c-shared -o client/libsrx_grpc_client.so  client/srx_grpc_client.go

cd ${GOPATH}/src/srx_grpc/version6/srx-server/src
make all install

cd ${GOPATH}/src/srx_grpc/version6/quagga-srx/bgpd
make all install


