package main

/*

#include <stdio.h>
#include "shared/srx_defs.h"
#include "client/srx_api.h"
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
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"time"
	"unsafe"
)

var port = flag.Int("port", 50000, "The server port")

var gStream_verify pb.SRxApi_ProxyVerifyStreamServer
var gBiStream_verify pb.SRxApi_ProxyVerifyBiStreamServer

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
var chProxySendAndWaitProcessData chan pb.PduResponse

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

	//cbVerifyNotify(int(f), b)

	cbService_VerifyNotify(int(f), b)

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

	log.Println("\033[0;32m++ [grpc server][cb_proxyStream](sync request) called from commandHandler as callback\033[0m")
	b := C.GoBytes(unsafe.Pointer(v), f)

	m := StreamData{
		data:   b,
		length: uint8(f),
	}
	log.Printf("\033[0;32m++ [grpc server][cb_proxyStream](sync request) sending Sync Request message to Channel: %#v\033[0m\n",
		m)

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
	log.Printf("\033[0;32m++ [grpc server][cb_proxyStream](sync request) is OVER  \033[0m \n")

}

//export cb_proxyCallbackHandler_Service
func cb_proxyCallbackHandler_Service(f C.int, v unsafe.Pointer) {
	log.Println("\033[0;32m++ [grpc server][cb_proxyCallbackHandler_Service]  General Purposes callback function\033[0m")

	b := C.GoBytes(unsafe.Pointer(v), f)

	go proxyCallbackHandler(int(f), b)

}

func proxyCallbackHandler(f int, b []byte) {
	fmt.Printf("++ [grpc server][proxyCallbackHandler]  received arg: %d, %#v \n", f, b)
	if f == 0 && b == nil {
		_, _, line, _ := runtime.Caller(0)
		log.Printf("++ [grpc server][:%d] length 0 - no data ", line)
		//done <- true
		return
	}
	var resp pb.PduResponse
	switch b[0] {
	case C.PDU_SRXPROXY_ERROR:
		resp = pb.PduResponse{
			Data:             b,
			Length:           uint32(len(b)),
			ValidationStatus: uint32(binary.BigEndian.Uint16(b[1:3])), // error code
		}
		/*
			typedef struct {
			  uint8_t     type;            // 11
			  uint16_t    errorCode;
			  uint8_t     reserved8;
			  uint32_t    zero32;
			  uint32_t    length;          // 12 Bytes
			} __attribute__((packed)) SRXPROXY_ERROR;
		*/
	}

	chProxySendAndWaitProcessData <- resp
	log.Printf("\033[0;32m++ [grpc server][proxyCallbackHandler] work done  \033[0m \n")
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

func cbService_VerifyNotify(f int, b []byte) {

	log.Printf("\033[0;32m++ [grpc server][cbService_VerifyNotify] gBiStream: %v - received arg: %d, %#v \033[0m \n",
		gBiStream_verify, f, b)
	var resp pb.ProxyVerifyNotify

	if gBiStream_verify != nil {

		if f == 0 && len(b) == 0 {
			_, _, line, _ := runtime.Caller(0)
			log.Printf("[server:%d] End of Notify", line)
			resp = pb.ProxyVerifyNotify{
				Type:   0,
				Length: 0,
			}

		} else {

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

		if err := gBiStream_verify.Send(&resp); err != nil {
			log.Printf("\033[0;32m++ [grpc server][cbService_VerifyNotify] grpc send error %#v  \033[0m \n", err)
		}
		_, _, line, _ := runtime.Caller(0)
		log.Printf("\033[0;32m++ [grpc server][cbService_VerifyNotify][server:%d] sending stream resp: %#v \033[0m \n",
			line+1, resp)

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

	log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] stream server received pdu: %s %#v \033[0m\n", pdu.Data, pdu)

	ctx := stream.Context()
	//ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	//defer cancel()
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		_, _, line, _ := runtime.Caller(0)
		log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess](:%d) server context done \033[0m\n", line+1)
		/*
			_, ok := <-done
			if ok == true {
				fmt.Printf("+ server close the channel done here\n")
				close(done)
			}
		*/
	}()

	/*
		//C.setLogMode(7)
			data := uint32(0x09)
			retData := C.RET_DATA{}
			retData = C.responseGRPC(C.int(pdu.Length), (*C.uchar)(unsafe.Pointer(&pdu.Data[0])), 0)

			b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
			fmt.Printf("return size: %d \t data: %#v\n", retData.size, b)

			resp := pb.PduResponse{
				Data:             b,
				Length:           uint32(retData.size),
				ValidationStatus: data,
			}
	*/

	for {
		log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] Waiting SendAndWaitProcess Channel Event ...\033[0m\n")
		select {
		case resp, ok := <-chProxySendAndWaitProcessData:
			if ok {
				log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] channel event resp: %#v \033[0m\n", resp)

				if err := stream.Send(&resp); err != nil {
					log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] Stream Terminated with the send error %v \033[0m\n", err)
					return err
				}
			} else {
				log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] Channel Closed and Stream terminated\033[0m\n")
				return nil
			}
		case <-ctx.Done():
			log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] Terminated due to context error \033[0m \n")
			return nil
		}
	}

	log.Printf("\033[0;31m++ [grpc server][SendAndWaitProcess] Finished with RPC send Send_Wait_Process...\033[0m\n")
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
				log.Printf("++ [grpc server][ProxyStream] Channel Closed and Stream terminated\n")
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

func (s *Server) ProxyVerifyBiStream(stream pb.SRxApi_ProxyVerifyBiStreamServer) error {
	log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream] stream: %#v \033[0m \n", stream)

	gBiStream_verify = stream // the function cbService_VerifyNotify() will use this variable for callback
	ctx := stream.Context()
	//ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	//defer cancel()

	for {

		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream] context error: %#v \033[0m \n", err)
				_, _, line, _ := runtime.Caller(0)
				log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream][:%d]context done \033[0m\n", line+1)
				return ctx.Err()
			}
		default:
		}

		log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream](RECV) WAITING for Receiving verify request...\033[0m\n")
		req, err := stream.Recv()

		if err == io.EOF {
			// return will close stream from server side
			log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream] exit (Error: %v) \033[0m \n", err)
			return nil
		}
		if err != nil {
			log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream] receive error %v \033[0m \n", err)
			continue
		}

		log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream](RECV) grpc ID: %02x, length: %d, \n Data: %#v \033[0m\n",
			req.GrpcClientID, req.Length, req.Data)

		retData := C.RET_DATA{}
		retData = C.responseGRPC(C.int(req.Length), (*C.uchar)(unsafe.Pointer(&req.Data[0])), C.uint(req.GrpcClientID))
		// retData comes from processValidationRequest_grpc()

		// only enabled once first notification received, in order not to have accumulated dealy
		b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
		log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream](SEND) VERIFY_NOTIFICATION_DATA .info: %x .size: %d \n .data: %#v \033[0m\n",
			retData.info, retData.size, b)

		// if sync resonpse case, retData should be NULL. So, no need to make notification
		if retData.size == 0 {
			continue
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
			log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream](SEDN) send error %#v \033[0m \n", err)
		}
		log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream](SEND) sending stream data: %#v \033[0m\n", resp)

		if retData.info == 0x1 {
			log.Printf("\033[1;33m++ [grpc server][ProxyVerifyBiStream](SEND) Notification send set with Queue  \033[0m\n")
			C.RunQueueCommand(C.int(req.Length), (*C.uchar)(unsafe.Pointer(&req.Data[0])), &retData, C.uint(req.GrpcClientID))
		}

	}

	return nil

}

func (s *Server) ProxyDeleteUpdate(ctx context.Context, req *pb.SerialPduDeleteUpdate) (*pb.PduResponse, error) {
	log.Printf("++ [grpc server][ProxyDeleteUpdate] grpc Client ID: %02x, data length: %d, \n Data: %#v\n",
		req.GrpcClientID, req.Length, req)
	/*
		this structure is the serialize data stream from client 'req'

			typedef struct {
			  uint8_t     type;            // 8
			  uint16_t    keepWindow;
			  uint8_t     reserved8;
			  uint32_t    zero32;
			  uint32_t    length;          // 16 Bytes
			  uint32_t    updateIdentifier;
			} __attribute__((packed)) SRXPROXY_DELETE_UPDATE;

				SRXPROXY_DELETE_UPDATE* hdr = malloc(length);
				hdr->type             = PDU_SRXPROXY_DELTE_UPDATE;
				hdr->keepWindow       = htons(keep_window);
				hdr->length           = htonl(length);
				hdr->updateIdentifier = htonl(updateID);
	*/
	//buf := make([]byte, 4) // uint32 updateIdentifier
	//binary.BigEndian.PutUint32(buf[0:4], pdu.data[12])

	retData := C.RET_DATA{}
	b := C.GoBytes(unsafe.Pointer(retData.data), C.int(retData.size))
	log.Printf("return size: %d \t data: %#v\n", retData.size, b)

	var updateId uint32
	updateId = *((*uint32)(unsafe.Pointer(&req.Data[12])))
	log.Printf("update ID: %#v\n", updateId)

	C.RunQueueCommand_uid(C.int(req.Length), (*C.uchar)(unsafe.Pointer(&req.Data[0])), C.uint32_t(updateId),
		C.uint(req.GrpcClientID))

	// Response should be any data
	data := uint32(0x01)
	return &pb.PduResponse{
		Data:             b,
		Length:           uint32(retData.size),
		ValidationStatus: data}, nil
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
func Serve(grpc_port C.int) {

	log_level := C.getLogLevel()
	log.Printf("srx server log level: %d\n", log_level)
	// NOTE: here init handling
	chGbsData = make(chan StreamData) // channel for Proxy GoodbyteStream
	chProxyStreamData = make(chan StreamData)
	chProxySendAndWaitProcessData = make(chan pb.PduResponse)

	// make a channel with a capacity of 100
	jobChan = make(chan Job, NUM_JobChan)

	log.Printf("++ [grpc server][Serve] received port number: %d \n", int32(grpc_port))

	flag.Parse()
	//lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", int32(grpc_port)))
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", int32(grpc_port)))
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}

	/* Disable Logging for performance measurement */
	if log_level < 6 {
		log.SetFlags(0)               // skip all formatting
		log.SetOutput(ioutil.Discard) // using this as io.Writer to skip logging
		os.Stdout = nil               // to suppress fmt.Print
	}

	server := NewServer(grpc.NewServer())
	if err := server.grpcServer.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}

func main() {
	flag.Parse()
	log.Printf("grpc server start (port:%d) ... \n", *port)
	Serve(C.int(*port))
}
