#!/bin/bash -x
LIBTOOL="/usr/bin/libtool --tag=CC"
ROOT="/opt/project/gobgp_test/gowork/src/srx_grpc_v6"
SRX_SERVER="srx-server"
QSRX="quagga-srx"
INSTALL_DIR=${ROOT}/_inst

cd ${ROOT}/${SRX_SERVER}/src
$LIBTOOL   --mode=compile gcc -DHAVE_CONFIG_H -g -O0 -c -o grpc_service.lo server/grpc_service.c \
    -I${ROOT}/${SRX_SERVER}/src \
    -I${ROOT}/${SRX_SERVER}/src/../extras/local/include \
    -I${ROOT}/${SRX_SERVER}/../_inst/include
$LIBTOOL   --mode=link   gcc -DHAVE_CONFIG_H -g -O0  -o libgrpc_service.la grpc_service.lo -rpath ${INSTALL_DIR}/lib64
$LIBTOOL   --mode=install cp libgrpc_service.la ${INSTALL_DIR}/lib64
$LIBTOOL   --finish ${INSTALL_DIR}/lib64

$LIBTOOL   --mode=compile gcc -DHAVE_CONFIG_H  -g -O0 -c -o grpc_client_service.lo client/grpc_client_service.c \
    -I${ROOT}/${SRX_SERVER}/src \
    -I${ROOT}/${SRX_SERVER}/src/../extras/local/include \
    -I${ROOT}/srx-server/../_inst/include
$LIBTOOL   --mode=link   gcc -DHAVE_CONFIG_H -g -O0  -o libgrpc_client_service.la grpc_client_service.lo -rpath ${INSTALL_DIR}/lib64
$LIBTOOL   --mode=install cp libgrpc_client_service.la ${INSTALL_DIR}/lib64
$LIBTOOL   --finish ${INSTALL_DIR}/lib64

cd ${ROOT}
go build -gcflags '-N -l' -buildmode=c-shared -o server/libsrx_grpc_server.so  server/srx_grpc_server.go
go build -gcflags '-N -l' -buildmode=c-shared -o client/libsrx_grpc_client.so  client/srx_grpc_client.go

cd ${ROOT}/${SRX_SERVER}/src
make all install

cd ${ROOT}/${QSRX}/bgpd
make all install


