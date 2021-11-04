package main

/*

#cgo CFLAGS: -DUSE_GRPC -g -Wall -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/../_inst/include/ -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/client -I/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/../extras/local/include

//#cgo LDFLAGS: /opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/.libs/log.o -L/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/.libs -lgrpc_service -Wl,-rpath -Wl,/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server//src//.libs -Wl,--unresolved-symbols=ignore-all

#cgo LDFLAGS: -L/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server/src/.libs -lgrpc_service -Wl,-rpath -Wl,/opt/project/gobgp_test/gowork/src/srx_grpc_v6/srx-server//src//.libs -Wl,--unresolved-symbols=ignore-all

#include <stdio.h>
#include "shared/srx_defs.h"
#include "srx/srx_api.h"
#include "server/grpc_service.h"



extern void cb_proxy(int f, void* user_data);
*/
import "C"

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
	pb "srx_grpc_v6"
	_ "sync"
	//	"github.com/golang/protobuf/proto"
	_ "bytes"
	"encoding/binary"
	_ "io"
	_ "io/ioutil"
	_ "os"
	"runtime"
	"time"
	"unsafe"
)

var port = flag.Int("port", 50000, "The server port")
var gStream pb.SRxApi_SendAndWaitProcessServer
var gStream_verify pb.SRxApi_ProxyVerifyStreamServer

//var gCancel context.CancelFunc

//var done chan bool

type Server struct {
	grpcServer *grpc.Server
}

type StreamData struct {
	data   []byte
	length uint8
}

var chGbsData chan StreamData
var chProxyStreamData chan StreamData

/* ---------- Worker Pool Start Block ---------------*/
/*
 * Worker goroutine Creation
 */
const NUM_JobChan = 1
const WorkerCount = 1

//var wg sync.WaitGroup
var jobChan chan Job

func worker(jobChan <-chan Job, workerId int32) {
	//defer wg.Done()

	log.Printf("++ [worker] (id: %d) goroutine generated and waiting job channel... \n ", workerId)
	// NOTE: The performance can be affected by this job channel's capacity
	// Because that will make the concurrency of Proxy Verify funtion
	/*
		for job := range jobChan {
			log.Printf("+++ [worker] (id: %d) job channel received : %#v\n", workerId, job)

			//ProxyVerify(job.data, job.id, job.done, workerId)
			chProxyStreamData <- job.data
			log.Printf("++ [worker] (id: %d) Sent StreamData to Channel  \n")
			log.Println("++ [worker] (id: %d) Waiting for the next job channel .... ")
		}
	*/

	job := <-jobChan
	log.Printf("++ [grpc server][worker](sync request) job: %#v\n", job)

	chProxyStreamData <- job.data

	// TODO: XXX  need to close job channel when all program done
	//			 - To prevent goroutine leaks
	log.Printf("++ [worker] (id: %d) goroutine closed \n ", workerId)
}

type Job struct {
	data StreamData
	id   uint32
	done chan bool
}

func NewJob(data StreamData, id uint32) *Job {

	log.Printf("+++ [grpc server][New Job](sync request) called \n ")
	// to prevent losing data slice, need to copy its slice into a new variable
	var d StreamData
	d = data

	return &Job{
		data: d,
		id:   id,
		done: make(chan bool),
	}
}

//export InitWorkerPool
func InitWorkerPool() bool {

	log.Printf("+++ [InitWorkerPool] go worker pool generating as many as worker counter: %d \n ", WorkerCount)
	// worker pool generation
	for i := 0; i < WorkerCount; i++ {
		//wg.Add(1)
		go worker(jobChan, int32(i))
	}

	//wg.Wait()
	log.Printf("+++ Init WorkerPool function closed  \n ")

	return true
}

/* ---------- Worker Pool End Block ---------------*/

//export cb_proxy
func cb_proxy(f C.int, v unsafe.Pointer) {
	/*
		fmt.Printf("++ [grpc server] proxy callback function : arg[%d, %#v]\n", f, v)
	*/

	b := C.GoBytes(unsafe.Pointer(v), f)

	// call my callback
	//TODO: distinguish two callback function

	//MyCallback(int(f), b)
	cbVerifyNotify(int(f), b)
}

//export cb_proxyGoodBye
func cb_proxyGoodBye(in C.SRXPROXY_GOODBYE) {

	GbIn := C.GoBytes(unsafe.Pointer(&in), C.sizeof_SRXPROXY_GOODBYE)

	m := StreamData{
		data:   GbIn,
		length: uint8(C.sizeof_SRXPROXY_GOODBYE),
	}
	log.Printf("channel callback message for server's GoodBye: %#v\n", m)

	chGbsData <- m
}

//export cb_proxyStream
func cb_proxyStream(f C.int, v unsafe.Pointer) {

	log.Println("++ [grpc server][cb_proxyStream](sync request) called from command handler function as callback")
	b := C.GoBytes(unsafe.Pointer(v), f)

	m := StreamData{
		data:   b,
		length: uint8(f),
	}
	log.Printf("++ [grpc server][cb_proxyStream](sync request) sending Sync Request message to Channel: %#v\n", m)

	/*
			job := NewJob(m, 1)

			log.Printf("++ [grpc server][cb_proxyStream](sync request) job: %#v\n", *job)

				select {
				case jobChan <- *job:
					log.Printf("++ [grpc server][cb_proxyStream](sync request) sending job channel and cb_proxyStream closed  \n")
					return
				}
		jobChan <- *job
	*/

	go func() {
		chProxyStreamData <- m
	}()
	//log.Printf("++ [grpc server][cb_proxyStream] Sent StreamData to Channel and callback function(cb_proxyStream) closed \n")
	log.Printf("++ [grpc server][cb_proxyStream](sync request) is OVER  \n")

}

func MyCallback(f int, b []byte) {

	fmt.Printf("++ [grpc server] My callback function - received arg: %d, %#v \n", f, b)

	if f == 0 && b == nil {
		_, _, line, _ := runtime.Caller(0)
		log.Printf("++ [grpc server][:%d] close stream ", line)
		//done <- true
		//return
	}

	//b := []byte{0x10, 0x11, 0x40, 0x42, 0xAB, 0xCD, 0xEF}
	resp := pb.PduResponse{
		Data:             b,
		Length:           uint32(len(b)),
		ValidationStatus: 2,
	}

	if gStream != nil {
		if resp.Data == nil && resp.Length == 0 {
			_, _, line, _ := runtime.Caller(0)
			log.Printf("++ [grpc server][:%d] close stream ", line)
			//close(done)
		} else {
			if err := gStream.Send(&resp); err != nil {
				log.Printf("send error %v", err)
			}
			_, _, line, _ := runtime.Caller(0)
			log.Printf("++ [grpc server][:%d] sending stream data", line+1)
		}

	}

}

func cbVerifyNotify(f int, b []byte) {
	/*
		fmt.Printf("++ [grpc server] [cbVerifyNotify] function - received arg: %d, %#v \n", f, b)
	*/
	var resp pb.ProxyVerifyNotify

	if gStream_verify != nil {

		if f == 0 && len(b) == 0 {
			_, _, line, _ := runtime.Caller(0)
			log.Printf("[server:%d] End of Notify", line)
			resp = pb.ProxyVerifyNotify{
				Type:   0,
				Length: 0,
			}
			//return gStream_verify.SendAndClose(&resp)

		} else {

			//TODO: length checking - if less than 16
			resp = pb.ProxyVerifyNotify{
				Type:         uint32(b[0]),
				ResultType:   uint32(b[1]),
				RoaResult:    uint32(b[2]),
				BgpsecResult: uint32(b[3]),
				AspaResult:   uint32(b[4]),
				Length:       *((*uint32)(unsafe.Pointer(&b[8]))),
				RequestToken: *((*uint32)(unsafe.Pointer(&b[12]))),
				UpdateID:     *((*uint32)(unsafe.Pointer(&b[16]))),
			}
		}

		if err := gStream_verify.Send(&resp); err != nil {
			log.Printf("[grpc server] grpc send error %#v", err)
		}
		_, _, line, _ := runtime.Caller(0)
		log.Printf("[server:%d] sending stream data", line+1)

	}

}

func (s *Server) SendPacketToSRxServer(ctx context.Context, pdu *pb.PduRequest) (*pb.PduResponse, error) {
	data := uint32(0x07)
	//C.setLogMode(3)
	fmt.Printf("server: %s %#v\n", pdu.Data, pdu)
	//C.setLogMode(7)
	fmt.Println("calling SRxServer responseGRPC()")

	retData := C.RET_DATA{}
	retData = C.responseGRPC(C.int(pdu.Length), (*C.uchar)(unsafe.Pointer(&pdu.Data[0])), 0)

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	fmt.Printf("return size: %d \t data: %#v\n", retData.size, b)

	//C.setLogMode(3)
	return &pb.PduResponse{
		Data:             b,
		Length:           uint32(retData.size),
		ValidationStatus: data}, nil
}

func (s *Server) SendAndWaitProcess(pdu *pb.PduRequest, stream pb.SRxApi_SendAndWaitProcessServer) error {

	gStream = stream
	ctx := stream.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	done := make(chan bool)
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}

		_, _, line, _ := runtime.Caller(0)
		fmt.Printf("+ [%d] server context done\n", line+1)

		close(done)
		// BUG NOTE : channel panic: close of closed channel
		/*
			_, ok := <-done
			if ok == true {
				fmt.Printf("+ server close the channel done here\n")
				close(done)
			}
		*/
	}()

	data := uint32(0x09)
	//C.setLogMode(3)
	fmt.Printf("stream server: %s %#v\n", pdu.Data, pdu)
	//C.setLogMode(7)
	fmt.Println("calling SRxServer responseGRPC()")

	retData := C.RET_DATA{}
	retData = C.responseGRPC(C.int(pdu.Length), (*C.uchar)(unsafe.Pointer(&pdu.Data[0])), 0)

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	fmt.Printf("return size: %d \t data: %#v\n", retData.size, b)

	resp := pb.PduResponse{
		Data:             b,
		Length:           uint32(retData.size),
		ValidationStatus: data,
	}

	if err := stream.Send(&resp); err != nil {
		log.Printf("send error %v", err)
	}
	log.Printf("sending stream data")

	//time.Sleep(5 * time.Second)

	<-done
	log.Printf("Finished with RPC send [Send_Wait_Process] \n")

	return nil
}

func (s *Server) ProxyHello(ctx context.Context, pdu *pb.ProxyHelloRequest) (*pb.ProxyHelloResponse, error) {
	//data := uint32(0x07)
	//C.setLogLevel(0x07)
	log.Printf("++ [grpc server][PorxyHello] input type :  %#v\n", pdu.Type)
	log.Printf("++ [grpc server][PorxyHello] received ProxyHelloRequest (size:%d): %#v \n", C.sizeof_SRXPROXY_HELLO, pdu)

	/* serialize */
	buf := make([]byte, C.sizeof_SRXPROXY_HELLO)
	buf[0] = byte(pdu.Type)
	binary.BigEndian.PutUint16(buf[1:3], uint16(pdu.Version))
	binary.BigEndian.PutUint32(buf[8:12], pdu.Length)
	binary.BigEndian.PutUint32(buf[12:16], pdu.ProxyIdentifier)
	binary.BigEndian.PutUint32(buf[16:20], pdu.Asn)
	binary.BigEndian.PutUint32(buf[20:24], pdu.NoPeerAS)

	grpcClientID := pdu.ProxyIdentifier

	//retData := C.RET_DATA{}
	log.Println("++ [grpc server][ProxyHello] Trying to call C.resonse-GRPC with CGO call ")
	retData := C.responseGRPC(C.int(C.sizeof_SRXPROXY_HELLO), (*C.uchar)(unsafe.Pointer(&buf[0])), C.uint(grpcClientID))

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	log.Printf("++ [grpc server][ProxyHello] return size: %d \t data: %#v\n", retData.size, b)

	return &pb.ProxyHelloResponse{
		Type:            uint32(b[0]),
		Version:         uint32(binary.BigEndian.Uint16(b[1:3])),
		Length:          binary.BigEndian.Uint32(b[8:12]),
		ProxyIdentifier: binary.BigEndian.Uint32(b[12:16]),
	}, nil
}

func (s *Server) ProxyGoodBye(ctx context.Context, pdu *pb.ProxyGoodByeRequest) (*pb.ProxyGoodByeResponse, error) {

	log.Println("++ [grpc server] calling SRxServer server:ProxyGoodBye()")
	log.Printf("++ [grpc server] input :  %#v\n", pdu.Type)
	log.Printf("++ [grpc server] ProxyGoodBye Request: %#v \n", pdu)

	/* serialize */
	buf := make([]byte, C.sizeof_SRXPROXY_GOODBYE)
	buf[0] = byte(pdu.Type)
	binary.BigEndian.PutUint16(buf[1:3], uint16(pdu.KeepWindow))
	buf[3] = byte(pdu.Reserved)
	binary.BigEndian.PutUint32(buf[4:8], pdu.Zero)
	binary.BigEndian.PutUint32(buf[8:12], pdu.Length)

	grpcClientID := pdu.GrpcClientID
	log.Printf("++ [grpc server] ProxyGoodBye grpcClientID : %02x \n", grpcClientID)

	retData := C.RET_DATA{}
	retData = C.responseGRPC(C.int(C.sizeof_SRXPROXY_GOODBYE), (*C.uchar)(unsafe.Pointer(&buf[0])),
		C.uint(grpcClientID))

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	log.Printf("++ [grpc server][ProxyGoodBye] return size: %d \t data: %#v\n", retData.size, b)

	return &pb.ProxyGoodByeResponse{
		Status: true,
	}, nil
}

func (s *Server) ProxyGoodByeStream(pdu *pb.PduRequest, stream pb.SRxApi_ProxyGoodByeStreamServer) error {
	log.Printf("++ [grpc server][ProxyGoodByeStream] pdu type: %02x \n", pdu.Data[0])
	log.Printf("++ [grpc server][ProxyGoodByeStream] received data: %#v\n", pdu)

	ctx := stream.Context()
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Printf("++ [grpc server][ProxyGoodByeStream] error : %#v \n", err)
		}

		_, _, line, _ := runtime.Caller(0)
		log.Printf("+ [%d] server Proxy_GoodBye_Stream context done\n", line+1)
		// XXX: panic - close a closed channel when run this program more than once, --> Do Not close
		//close(chGbsData)
		return
	}()

	log.Printf("++ [grpc server][ProxyGoodByeStream] Waiting Channel Event ...\n")
	for {
		select {
		case m, ok := <-chGbsData:
			if ok {
				log.Printf("channel event message : %#v\n", m)
				resp := pb.PduResponse{
					Data:   m.data,
					Length: uint32(len(m.data)),
				}

				if err := stream.Send(&resp); err != nil {
					log.Printf("send error %v", err)
					return err
				}
			} else {
				log.Printf("++ [grpc server][ProxyGoodByeStream] Channel Closed\n")
				// TODO: instead of nil, it should have error value returned
				//		How To define Error ?
				return nil
			}
		}
	}

	return nil
}

func (s *Server) ProxyStream(pdu *pb.PduRequest, stream pb.SRxApi_ProxyStreamServer) error {
	log.Printf("++ [grpc server][ProxyStream] pdu type: %02x \n", pdu.Data[0])
	log.Printf("++ [grpc server][ProxyStream] received data: %#v\n", pdu)

	ctx := stream.Context()
	//ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	//defer cancel()
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Printf("++ [grpc server][ProxyStream] context error: %#v \n", err)
		}
		_, _, line, _ := runtime.Caller(0)
		log.Printf("++ [grpc server][ProxyStream][:%d] (goroutine close) server Proxy_Stream context done\n", line+1)
		return
	}()

	log.Printf("++ [grpc server][ProxyStream] Waiting Channel Event ...\n")
	for {
		select {
		case m, ok := <-chProxyStreamData:
			if ok {
				log.Printf("++ [grpc server][ProxyStream] channel event message : %#v\n", m)
				resp := pb.PduResponse{
					Data:   m.data,
					Length: uint32(len(m.data)),
				}
				log.Printf("++ [grpc server][ProxyStream] Sending sync request data to client proxy ...\n")
				if err := stream.Send(&resp); err != nil {
					log.Printf("++ [grpc server][ProxyStream] Stream Terminated with the send error %v \n", err)
					return err
				}
			} else {
				log.Printf("++ [grpc server][ProxyGoodByeStream] Channel Closed and Stream terminated\n")
				// TODO: instead of nil, it should have error value returned
				//		How To define Error ?
				return nil
			}
		case <-ctx.Done():
			log.Printf("++ [grpc server][ProxyStream] Terminated due to context error \n")
			return nil

		}
	}
	log.Printf("++ [grpc server][ProxyStream] Terminated ... \n")
	return nil
}

// stale function -- depricated
func (s *Server) ProxyVerify(pdu *pb.ProxyVerifyV4Request, stream pb.SRxApi_ProxyVerifyServer) error {
	fmt.Println("calling SRxServer server:ProxyVerify()")

	gStream_verify = stream
	ctx := stream.Context()
	done := make(chan bool)
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(done)
	}()

	fmt.Printf("stream server: %#v\n", pdu)

	retData := C.RET_DATA{}
	retData = C.responseGRPC(C.int(0), (*C.uchar)(unsafe.Pointer(nil)), 0)

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	fmt.Printf("return size: %d \t data: %#v\n", retData.size, b)

	resp := pb.ProxyVerifyNotify{
		Type:       0,
		ResultType: 0,
		RoaResult:  0,
	}

	if err := stream.Send(&resp); err != nil {
		log.Printf("send error %v", err)
	}
	log.Printf("sending stream data")

	//time.Sleep(5 * time.Second)

	<-done
	log.Printf("Finished with RPC send \n")

	return nil
}

func (s *Server) ProxyVerifyStream(pdu *pb.ProxyVerifyRequest, stream pb.SRxApi_ProxyVerifyStreamServer) error {
	log.Println("++ [grpc server] calling SRxServer server:ProxyVerifyStream()")

	gStream_verify = stream // the function cbVerifyNotify() will this variable for callback
	ctx := stream.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	done := make(chan bool)
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Printf("++ [grpc server][ProxyVerifyStream] context error: %#v \n", err)
		}
		_, _, line, _ := runtime.Caller(0)
		log.Printf("++ [grpc server][ProxyVerifyStream][:%d]context done\n", line+1)
		close(done)
	}()

	log.Printf("++ [grpc server][ProxyVerifyStream] grpc Client ID: %02x, data length: %d, \n Data: %#v\n",
		pdu.GrpcClientID, pdu.Length, pdu)

	log.Println("++ [grpc server][ProxyVerifyStream] calling SRxServer responseGRPC()")

	retData := C.RET_DATA{}
	retData = C.responseGRPC(C.int(pdu.Length), (*C.uchar)(unsafe.Pointer(&pdu.Data[0])), C.uint(pdu.GrpcClientID))

	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))

	log.Printf("++ [grpc server][ProxyVerifyStream] return size: %d \t data: %#v\n", retData.size, b)

	if retData.size == 0 {
		return nil
	}

	resp := pb.ProxyVerifyNotify{
		Type:         uint32(b[0]),
		ResultType:   uint32(b[1]),
		RoaResult:    uint32(b[2]),
		BgpsecResult: uint32(b[3]),
		AspaResult:   uint32(b[4]),
		Length:       *((*uint32)(unsafe.Pointer(&b[8]))),
		RequestToken: *((*uint32)(unsafe.Pointer(&b[12]))),
		UpdateID:     *((*uint32)(unsafe.Pointer(&b[16]))),
	}

	if err := stream.Send(&resp); err != nil {
		log.Printf("send error %#v", err)
	}
	log.Printf("++ [grpc server][ProxyVerifyStream] sending stream data")

	<-done
	log.Printf("++ [grpc server][ProxyVerifyStream] Finished with RPC send \n")

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

	// NOTE: here init handling
	chGbsData = make(chan StreamData) // channel for Proxy GoodbyteStream
	chProxyStreamData = make(chan StreamData)

	// make a channel with a capacity of 100
	jobChan = make(chan Job, NUM_JobChan)

	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}

	/* Disable Logging for performance measurement */
	/*
		log.SetFlags(0)               // skip all formatting
		log.SetOutput(ioutil.Discard) // using this as io.Writer to skip logging
		os.Stdout = nil               // to suppress fmt.Print
	*/

	server := NewServer(grpc.NewServer())
	if err := server.grpcServer.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}

func main() {
	log.Println("grpc server start ... ")
	Serve()
}
