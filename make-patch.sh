#!/bin/bash 
ROOT="/opt/project/gobgp_test/gowork/src/srx_grpc_v6"
SRX_SERVER="srx-server"
QSRX="quagga-srx"
SUBMODULE="NIST-BGP-SRx"

SRX_PATCH="diff_grpc_srx-server_github_v614-temp.patch"
QSRX_PATCH="diff_grpc_quagga-srx_github_v614-temp.patch"



cd ${ROOT}/${SRX_SERVER}/src
find ./ \( -name '*.[ch]' -o -name '*.am' -o -name '*.ac' -o -name '*.conf' \) ! -name 'config.h' ! -name 'srxproxy64.conf' -exec diff -wibuNp ${ROOT}/${SUBMODULE}/${SRX_SERVER}/src/{} {} \; > ${ROOT}/${SRX_PATCH}

#sed 's/\/opt\/project\/gobgp_test\/gowork\/src\/srx_grpc_v6\/NIST-BGP-SRx\/quagga-srx\///' diff_grpc_quagga-srx_github_v614-temp.patch

cd ${ROOT}/${QSRX}
find ./ \( -name '*.[ch]' -o -name '*.am' -o -name '*.ac' \) ! -name 'config.h' ! -name 'version.h' -exec diff -wibuNp ${ROOT}/${SUBMODULE}/${QSRX}/{} {} \; > ${ROOT}/${QSRX_PATCH}



#find ./ \( -name '*.[ch]' -o -name '*.am' -o -name '*.ac' -o -name '*.conf' \) ! -name 'config.h' ! -name 'srxproxy64.conf' -exec diff -wibuNp /opt/project/gobgp_test/gowork/src/srx_grpc_v6/NIST-BGP-SRx/srx-server/src/{} {} \; > temp1

#find ./ \( -name '*.[ch]' -o -name '*.am' -o -name '*.ac' \) ! -name 'config.h' ! -name 'version.h' -exec diff -wibuNp /opt/project/gobgp_test/gowork/src/srx_grpc_v6/NIST-BGP-SRx/quagga-srx/{} {} \; > diff_grpc_quagga-srx-github-v2.patch


#sed 's/\/opt\/project\/gobgp_test\/gowork\/src\/srx_grpc_v6\/NIST-BGP-SRx\/quagga-srx\///' diff_grpc_quagga-srx_github_v614-temp.patch
