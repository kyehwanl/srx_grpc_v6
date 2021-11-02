package main

/*

#cgo CFLAGS: -I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/test_install/include/ -I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/ -I/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/../extras/local/include -I/opt/project/srx_test1/srx/../_inst//include
#cgo LDFLAGS: -L/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/server -lgrpc_service -Wl,-rpath -Wl,/opt/project/gobgp_test/gowork/src/srx_grpc/srx/src/server -L/opt/project/gobgp_test/gowork/src/srx_grpc/srx/test_install/lib64/srx -lSRxProxy -Wl,-rpath -Wl,/opt/project/gobgp_test/gowork/src/srx_grpc/srx/test_install/lib64/srx -Wl,--unresolved-symbols=ignore-all

#include <stdio.h>
#include "srx/srx_api.h"
#include "server/grpc_service.h"
#include <stdio.h>
*/
import "C"

import (
	"flag"
	"fmt"
	"log"
	"net"
	pb "srx_grpc"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	//	"github.com/golang/protobuf/proto"
	_ "bytes"
	"time"
	"unsafe"
)

var port = flag.Int("port", 50000, "The server port")

type Server struct {
	grpcServer *grpc.Server
}

func (s *Server) SendPacketToSRxServer(ctx context.Context, pdu *pb.PduRequest) (*pb.PduResponse, error) {
	data := uint32(0x07)
	fmt.Printf("server: %s %#v\n", pdu.Data, pdu)

	gostr := "12345ABCD SendPacketToSRxServer"
	cstr := C.CString(gostr)

	retData := C.RET_DATA{
		size: C.uint(len(gostr)),
		data: (*C.uchar)(unsafe.Pointer(cstr)),
	}
	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	fmt.Printf("return size: %d \t data: %#v\n", retData.size, b)

	return &pb.PduResponse{
		Data:             b,
		Length:           uint32(retData.size),
		ValidationStatus: data}, nil
}

func (s *Server) SendAndWaitProcess(pdu *pb.PduRequest, stream pb.SRxApi_SendAndWaitProcessServer) error {

	data := uint32(0x09)
	fmt.Printf("stream server: %s %#v\n", pdu.Data, pdu)

	gostr := "sS 123456789 abcdef - SendAndWaitProcess"
	cstr := C.CString(gostr)

	retData := C.RET_DATA{
		size: C.uint(len(gostr)),
		data: (*C.uchar)(unsafe.Pointer(cstr)),
	}

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	fmt.Printf("return size: %d \t data: %#v\n", retData.size, b)

	resp := pb.PduResponse{
		Data:             b,
		Length:           uint32(retData.size),
		ValidationStatus: data,
	}

	// send test 1
	if err := stream.Send(&resp); err != nil {
		log.Printf("send error %v", err)
	}
	log.Printf("+ send stream data")

	time.Sleep(2 * time.Second)

	// send test 2
	if err := stream.Send(&resp); err != nil {
		log.Printf("send error %v", err)
	}
	log.Printf("++ send stream data - one more time")

	return nil
}

func NewServer(g *grpc.Server) *Server {
	grpc.EnableTracing = false
	server := &Server{
		grpcServer: g,
	}
	pb.RegisterSRxApiServer(g, server)
	return server
}

//export Serve
func Serve() {

	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := NewServer(grpc.NewServer())
	if err := server.grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	Serve()
}
