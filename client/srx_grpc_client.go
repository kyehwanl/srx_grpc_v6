package main

/*

#include <stdlib.h>
#include "shared/srx_packets.h"
#include "client/grpc_client_service.h"
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	pb "srx_grpc_v6"
	_ "sync"
	"time"
	"unsafe"
)

const (
	address     = "localhost:50000"
	defaultName = "RPKI_DATA"
)

const NUM_PREFIX = 100000
const NUM_JobChan = 1000

type Client struct {
	conn *grpc.ClientConn
	cli  pb.SRxApiClient
}

var client Client

type ProxyVerifyClient struct {
	stream pb.SRxApi_ProxyVerifyClient
}

var g_count int32
var start time.Time
var elapsed time.Duration
var g_std *os.File
var gw http.ResponseWriter

//var wg sync.WaitGroup
//var wgVf sync.WaitGroup
var jobChan chan Job
var chDoneProxyHello chan bool
var chVerifyData chan Job

const WorkerCount = 1

var gCount int32

func worker(jobChan <-chan Job, workerId int32) {
	//defer wg.Done()

	// NOTE: The performance can be affected by this job channel's capacity
	// Because that will make the concurrency of Proxy Verify funtion
	for job := range jobChan {
		job.workerId = workerId

		log.Printf("+++ [worker] (id: %d) job channel received : %v\n", workerId, job)
		log.Println("+++ start Proxy Verify")

		//ProxyVerify(job.data, job.grpcClientID, job.done, workerId)
		chVerifyData <- job

		log.Println("+++ Finished Proxy Verify ")
		log.Println("+++ ... Waiting for the next job channel .... ")

	}

	// TODO: XXX  need to close job channel when all program done
	//			 - To prevent goroutine leaks

	log.Printf("+++ worker(id: %d) goroutine closed \n ", workerId)
}

type Job struct {
	data         []byte
	grpcClientID uint32
	workerId     int32
	done         chan bool
}

func NewJob(data []byte, grpcClientID uint32) *Job {

	log.Printf("+++ [New Job] called \n ")
	// to prevent losing data slice, need to copy its slice into a new variable
	var d = make([]byte, len(data))
	copy(d, data)

	return &Job{
		data:         d,
		grpcClientID: grpcClientID,
		done:         make(chan bool),
	}
}

//export InitWorkerPool
func InitWorkerPool() bool {

	log.Printf("+++ [InitWorkerPool] go worker pool generating as many as worker counter: %d \n ", WorkerCount)
	// worker pool generation
	for i := 0; i < WorkerCount; i++ {
		//wg.Add(1)
		gCount++
		go worker(jobChan, gCount)
	}

	//wg.Wait()
	log.Printf("+++ Init WorkerPool function closed  \n ")

	return true
}

//export InitSRxGrpc
func InitSRxGrpc(addr string) bool {

	/* Disable Logging */
	/*
		log.SetFlags(0)               // skip all formatting
		log.SetOutput(ioutil.Discard) // using this as io.Writer to skip logging. To restore, use os.Stdout
		g_std = os.Stdout             // backup for later use
		os.Stdout = nil               // to suppress fmt.Print
	*/

	log.Printf("[InitSRxGrpc] InitSRxGrpc Called \n")
	//conn, err := grpc.Dial(addr, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	//conn, err := grpc.Dial(addr, grpc.WithBlock())
	if err != nil {
		log.Printf("[InitSRxGrpc] did not connect: %v", err)
		return false
	}
	//log.Printf("conn: %#v, err: %#v\n", conn, err)
	log.Printf("[InitSRxGrpc] gRPC Client Initiated and Connected Server Address: %s\n", addr)
	//defer conn.Close()
	cli := pb.NewSRxApiClient(conn)

	client.conn = conn
	client.cli = cli

	// make a channel with a capacity of 100
	jobChan = make(chan Job, NUM_JobChan)
	chDoneProxyHello = make(chan bool)

	// input data for verify update
	chVerifyData = make(chan Job)

	log.Printf("[InitSRxGrpc] worker pool init \n")
	InitWorkerPool()

	go ImpleProxyVerifyBiStream()
	go ImpleSendAndWaitProcessStream(nil)

	//fmt.Printf("cli : %#v\n", cli)
	//fmt.Printf("client.cli : %#v\n", client.cli)
	//fmt.Println()
	return true
}

//export Run
func Run(data []byte) uint32 {
	// Set up a connection to the server.
	cli := client.cli
	fmt.Printf("client : %#v\n", client)
	fmt.Printf("client.cli data: %#v\n", client.cli)
	fmt.Println()

	// Contact the server and print out its response.
	//ctx, _ := context.WithTimeout(context.Background(), time.Second)
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()

	if data == nil {
		data = []byte(defaultName)
	}

	fmt.Printf("input data: %#v\n", data)

	r, err := cli.SendPacketToSRxServer(context.Background(), &pb.PduRequest{Data: data, Length: uint32(len(data))})
	if err != nil {
		log.Printf("could not receive: %v", err)
	}

	fmt.Printf("data : %#v\n", r.Data)
	fmt.Printf("size : %#v\n", r.Length)
	fmt.Printf("status: %#v\n", r.ValidationStatus)
	fmt.Println()

	//return r.ValidationStatus, err
	return uint32(r.ValidationStatus)
}

//export RunProxyHello
func RunProxyHello(data []byte) (*C.uchar, uint32) {
	/*
		defer func() {
			log.Println("+ [RunProxyHello] just before sending a data to channel  ")
			chDoneProxyHello <- true
			log.Println("+ [RunProxyHello] just after sending a data to channel  ")
		}()
	*/

	log.Println("+ [RunProxyHello] called in srx_grpc_server.go  ")
	cli := client.cli
	fmt.Println()
	fmt.Printf("+ [RunProxyHello] client : %#v\n", client)
	fmt.Printf("+ [RunProxyHello] client.cli : %#v\n", client.cli)
	//fmt.Println(cli)
	fmt.Printf("+ [RunProxyHello] input data: %#v\n", data)
	/*
		retData := C.RET_DATA{}
		hData := C.SRXPROXY_HELLO{}
		fmt.Println("temp:", retData, hData)
	*/

	req := pb.ProxyHelloRequest{
		Type:            uint32(data[0]),
		Version:         uint32(binary.BigEndian.Uint16(data[1:3])),
		Reserved:        uint32(data[3]),
		Zero:            binary.BigEndian.Uint32(data[4:8]),
		Length:          binary.BigEndian.Uint32(data[8:12]),
		ProxyIdentifier: binary.BigEndian.Uint32(data[12:16]),
		Asn:             binary.BigEndian.Uint32(data[16:20]),
		NoPeerAS:        binary.BigEndian.Uint32(data[20:24]),
	}

	log.Println("+ [grpc client][RunProxyHello] Trying to call cli.ProxyHello grpc context  through protocol buffer \n")
	resp, err := cli.ProxyHello(context.Background(), &req)
	if err != nil {
		log.Printf("+ [grpc client][RunProxyHello] Error - could not receive: (%v)\n", err)
	}

	log.Printf("+ [grpc client][RunProxyHello] HelloRequest	: %#v\n", req)
	log.Printf("+ [grpc client][RunProxyHello] response		: %#v\n", resp)

	rp := C.SRXPROXY_HELLO_RESPONSE{
		//version:         C.ushort(resp.Version), // --> TODO: need to pack/unpack for packed struct in C
		//version:         C.ushort(resp.Version), // --> TODO: need to pack/unpack for packed struct in C
		length:          C.uint(resp.Length),
		proxyIdentifier: C.uint(resp.ProxyIdentifier),
	}
	rp._type = C.uchar(resp.Type)

	//fmt.Println("rp:", rp)

	buf := make([]byte, C.sizeof_SRXPROXY_HELLO_RESPONSE)
	//buf := make([]byte, 12)
	buf[0] = byte(resp.Type)
	binary.BigEndian.PutUint16(buf[1:3], uint16(resp.Version))
	binary.BigEndian.PutUint32(buf[8:12], resp.Length)
	binary.BigEndian.PutUint32(buf[12:16], resp.ProxyIdentifier)

	cb := (*[C.sizeof_SRXPROXY_HELLO_RESPONSE]C.uchar)(C.malloc(C.sizeof_SRXPROXY_HELLO_RESPONSE))
	// TODO: defer C.free(unsafe.Pointer(cb)) at caller side --> DONE
	cstr := (*[C.sizeof_SRXPROXY_HELLO_RESPONSE]C.uchar)(unsafe.Pointer(&buf[0]))

	for i := 0; i < C.sizeof_SRXPROXY_HELLO_RESPONSE; i++ {
		cb[i] = cstr[i]
	}

	log.Printf("+ [grpc client][RunProxyHello] Received Hello Response message: %#v\n", cb)
	//return (*C.uchar)(unsafe.Pointer(&buf[0]))
	return &cb[0], resp.ProxyIdentifier
}

type Go_PduProxyError struct {
	_type      uint8
	_errorCode uint16
	_reserved  uint8
	_zero      uint32
	_length    uint32
}

func (g *Go_PduProxyError) Pack(out unsafe.Pointer) {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, g)
	l := buf.Len()
	o := (*[1 << 20]C.uchar)(out)

	for i := 0; i < l; i++ {
		b, _ := buf.ReadByte()
		o[i] = C.uchar(b)
	}
}

type Go_ProxySyncRequest struct {
	_type     uint8
	_reserved uint16
	_reserve2 uint8
	_zero     uint32
	_length   uint32
}

func (g *Go_ProxySyncRequest) Pack(out unsafe.Pointer) {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, g)
	l := buf.Len()
	o := (*[1 << 20]C.uchar)(out)

	for i := 0; i < l; i++ {
		b, _ := buf.ReadByte()
		o[i] = C.uchar(b)
	}
}

type Go_ProxyVerifyNotify struct {
	_type         uint8
	_resultType   uint8
	_roaResult    uint8
	_bgpsecResult uint8
	_aspaResult   uint8
	_reserve      uint8
	_zero         uint16
	_length       uint32
	_requestToken uint32
	_updateID     uint32
}

func (g *Go_ProxyVerifyNotify) Pack(out unsafe.Pointer) {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, g)
	l := buf.Len()
	o := (*[1 << 30]C.uchar)(out)

	for i := 0; i < l; i++ {
		b, _ := buf.ReadByte()
		o[i] = C.uchar(b)
	}
}

type Go_ProxyGoodBye struct {
	_type       uint8
	_keepWindow uint16
	_reserved   uint8
	_zero       uint32
	_length     uint32
}

func (g *Go_ProxyGoodBye) Pack(out unsafe.Pointer) {

	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, g)
	l := buf.Len()
	o := (*[1 << 20]C.uchar)(out)

	for i := 0; i < l; i++ {
		b, _ := buf.ReadByte()
		o[i] = C.uchar(b)
	}
}

func (g *Go_ProxyGoodBye) Unpack(i *C.SRXPROXY_GOODBYE) {

	cdata := C.GoBytes(unsafe.Pointer(i), C.sizeof_SRXPROXY_GOODBYE)
	buf := bytes.NewBuffer(cdata)
	binary.Read(buf, binary.BigEndian, &g._type)
	binary.Read(buf, binary.BigEndian, &g._keepWindow)
	binary.Read(buf, binary.BigEndian, &g._reserved)
	binary.Read(buf, binary.BigEndian, &g._zero)
	binary.Read(buf, binary.BigEndian, &g._length)
}

//export RunProxyGoodBye
func RunProxyGoodBye(in C.SRXPROXY_GOODBYE, grpcClientID uint32) bool {
	cli := client.cli

	log.Printf("++ [grpc client][RunProxyGoodBye] Goobye function: input parameter: %#v \n", in)
	log.Printf("++ [grpc client][RunProxyGoodBye] Goobye function: size: %d \n", C.sizeof_SRXPROXY_GOODBYE)

	goGB := Go_ProxyGoodBye{}
	goGB.Unpack(&in)
	//out := (*[C.sizeof_SRXPROXY_GOODBYE]C.uchar)(C.malloc(C.sizeof_SRXPROXY_GOODBYE))
	log.Printf("++ [grpc client][RunProxyGoodBye] Goodbye out bytes: %#v\n", goGB)

	req := pb.ProxyGoodByeRequest{
		Type:         uint32(goGB._type),
		KeepWindow:   uint32(goGB._keepWindow),
		Reserved:     uint32(goGB._reserved),
		Zero:         uint32(goGB._zero),
		Length:       uint32(goGB._length),
		GrpcClientID: grpcClientID,
	}
	log.Printf("++ [grpc client][RunProxyGoodBye] sending GoodByeRequest	: %#v\n", req)
	resp, err := cli.ProxyGoodBye(context.Background(), &req)
	if err != nil {
		log.Printf("could not receive: %v", err)
		return false
	}
	log.Printf("++ [grpc client][RunProxyGoodBye] The received GoodBye response: %#v\n", resp)
	log.Printf("++ [grpc client][RunProxyGoodBye] Function RunProxyGoodBye Done\n")

	return resp.Status
}

//export RunProxyGoodByeStream
func RunProxyGoodByeStream(data []byte, grpcClientID uint32) uint32 {
	defer close(jobChan)

	//fmt.Printf("+ [grpc client] Goobye Stream function Started : input parameter: %#v \n", data)
	cli := client.cli
	stream, err := cli.ProxyGoodByeStream(context.Background(), &pb.PduRequest{Data: data, Length: uint32(len(data))})
	ctx := stream.Context()
	if err != nil {
		log.Printf("open stream error %v", err)
	}

	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Printf("+ [grpc client][GoodByeStream] context error: %v", err)
		}
		log.Printf("+ [grpc client][GoodByeStream] client context done\n")
		return
	}()

	resp, err := stream.Recv()
	log.Printf("+ [grpc client][GoodByeStream] Goodbye Stream Received from the server... \n")
	if err == io.EOF {
		log.Printf("+ [grpc client][GoodByeStream] EOF close \n")
		C.processGoodbye_grpc(nil)
		return 1
	}
	if err != nil {
		log.Printf("+ [grpc client][GoodByeStream] ERROR (%v)\n", err)
		C.processGoodbye_grpc(nil)
		return 1
	}

	// NOTE : receive process here
	log.Printf("+ [grpc client][GoodByeStream] data : %#v\n", resp.Data)
	log.Printf("+ [grpc client][GoodByeStream] size : %#v\n", resp.Length)
	fmt.Println()

	go_gb := &Go_ProxyGoodBye{
		_type:       resp.Data[0],
		_keepWindow: *((*uint16)(unsafe.Pointer(&resp.Data[1]))),
		_reserved:   resp.Data[3],
		_zero:       *((*uint32)(unsafe.Pointer(&resp.Data[4]))),
		_length:     *((*uint32)(unsafe.Pointer(&resp.Data[8]))),
	}

	gb := (*C.SRXPROXY_GOODBYE)(C.malloc(C.sizeof_SRXPROXY_GOODBYE))
	defer C.free(unsafe.Pointer(gb))
	go_gb.Pack(unsafe.Pointer(gb))

	log.Printf("+ [grpc client][GoodByeStream] received goodbye resopnse data: %#v\n", gb)

	//void processGoodbye_grpc(SRXPROXY_GOODBYE* hdr)
	C.processGoodbye_grpc(gb)

	return 0
}

//export RunProxyStream
func RunProxyStream(data []byte, grpcClientID uint32) uint32 {
	/*
	 This function is to deal with PDU_SRXPROXY_SYNC_REQUEST,
	    PDU_SRXPROXY_SIGN_NOTIFICATION and so on
	*/

	log.Printf("+ [grpc client][ProxyStream] Stream function Started : input parameter: %#v \n", data)
	cli := client.cli
	stream, err := cli.ProxyStream(context.Background(), &pb.PduRequest{Data: data, Length: uint32(len(data))})
	ctx := stream.Context()
	if err != nil {
		log.Printf("+ [grpc client][ProxyStream] open stream error %v", err)
	}

	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Printf("+ [grpc client][ProxyStream] context error: %v", err)
		}
		log.Printf("+ [grpc client][ProxyStream] client context done\n")
		return
	}()

	log.Printf("+ [grpc client][ProxyStream] Waiting for Recv... \n")
	resp, err := stream.Recv()
	log.Printf("+ [grpc client][ProxyStream] Proxy_Stream function Received... \n")
	if err == io.EOF {
		log.Printf("+ [grpc client][ProxyStream] EOF close\n ")
		return 1
	}
	if err != nil {
		log.Printf("+ [grpc client][ProxyStream] Error (%v)\n", err)
		return 1
	}

	// NOTE : receive process here
	//<-chDoneProxyHello
	log.Printf("+ [grpc client][ProxyStream] data : %#v\n", resp.Data)
	log.Printf("+ [grpc client][ProxyStream] size : %#v\n", resp.Length)
	log.Println()

	// NOTE: time delay for making sure to finish the code who called ProxyHello,
	// which is conntectToSRx_grpc function to set the flags(connHandler->initilaized and established)
	// before C.processSyncRequest entering to verify_update() to prevent from accessing
	// initialized and established flgas in isConnected(bgp->proxy) routine
	<-time.After(2 * time.Second)

	if resp.Data == nil || resp.Length == 0 {
		_, _, line, _ := runtime.Caller(0)
		log.Printf("+ [grpc client][ProxyStream][line:%d] not available message", line+1)

	} else {

		switch resp.Data[0] {
		case C.PDU_SRXPROXY_SYNC_REQUEST:
			log.Printf("+ [grpc client][ProxyStream] Sync Request Received from server\n")

			go_sr := &Go_ProxySyncRequest{
				_type:     resp.Data[0],
				_reserved: *((*uint16)(unsafe.Pointer(&resp.Data[1]))),
				_reserve2: resp.Data[3],
				_zero:     *((*uint32)(unsafe.Pointer(&resp.Data[4]))),
				_length:   *((*uint32)(unsafe.Pointer(&resp.Data[8]))),
			}
			sr := (*C.SRXPROXY_SYNCH_REQUEST)(C.malloc(C.sizeof_SRXPROXY_SYNCH_REQUEST))
			defer C.free(unsafe.Pointer(sr))
			go_sr.Pack(unsafe.Pointer(sr))
			log.Printf("+ [grpc client][ProxyStream] received sync request message: %#v\n", sr)

			//void processSyncRequest_grpc(SRXPROXY_SYNCH_REQUEST* hdr)
			C.processSyncRequest_grpc(sr)

		case C.PDU_SRXPROXY_SIGN_NOTIFICATION:
			log.Printf("+ [grpc client][ProxyStream] Sign Notification\n")

			// TODO: XXX need to supplement below
			//void processSignNotify_grpc(SRXPROXY_SIGNATURE_NOTIFICATION* hdr)
			C.processSignNotify_grpc(nil)

		}
	}

	log.Printf("+ [grpc client][ProxyStream] Terminated ...\n")

	return 0
}

//export ImpleSendAndWaitProcessStream
func ImpleSendAndWaitProcessStream(data []byte) uint32 {

	cli := client.cli
	log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] client data: %#v\n", client)

	if data == nil {
		fmt.Println("#############")
		data = []byte(defaultName)
	}

	log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] input data for stream response: %#v\033[0m\n", data)

	stream, err := cli.SendAndWaitProcess(context.Background(), &pb.PduRequest{Data: data, Length: uint32(len(data))})
	if err != nil {
		log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] open stream error %v \033[0m \n", err)
	}

	ctx := stream.Context()
	done := make(chan bool)
	//var r pb.PduResponse

	go func() {
		defer close(done)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] EOF close \033[0m \n")
				return
			}
			if err != nil {
				log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] can not receive %v \033[0m \n", err)
				return
			}

			// NOTE : receive process here
			log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] data  : %#v\033[0m\n", resp.Data)
			log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] size  : %#v\033[0m\n", resp.Length)
			log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] status: %#v\033[0m\n", resp.ValidationStatus)
			//r = resp

			if resp.Data == nil && resp.Length == 0 {
				_, _, line, _ := runtime.Caller(0)
				log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess](:%d) close stream \033[0m \n", line+1)
				//done <- true
				//stream.CloseSend()
			} else {

				switch resp.Data[0] {
				case C.PDU_SRXPROXY_ERROR:
					log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] Error Message Received from server\033[0m\n")
					go_ppe := &Go_PduProxyError{
						_type:      resp.Data[0],
						_errorCode: *((*uint16)(unsafe.Pointer(&resp.Data[1]))),
						_reserved:  resp.Data[3],
						_zero:      *((*uint32)(unsafe.Pointer(&resp.Data[4]))),
						_length:    *((*uint32)(unsafe.Pointer(&resp.Data[8]))),
					}
					ppe := (*C.SRXPROXY_ERROR)(C.malloc(C.sizeof_SRXPROXY_ERROR))
					defer C.free(unsafe.Pointer(ppe))
					go_ppe.Pack(unsafe.Pointer(ppe))

					//  call C function in client implementations
					//  callback to the client to transfer the server's message
					C.processError_grpc(ppe)
				}

			}
		} // end of for loop

	}()

	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println("\033[0;31m+ [grpc client][SendAndWaitProcess]\033[0m", err)
		}
		log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] client context done\033[0m\n")
	}()

	<-done
	//log.Printf("Finished with Resopnse valie: %d", uint32(resp.ValidationStatus))
	log.Printf("\033[0;31m+ [grpc client][SendAndWaitProcess] Finished \033[0m\n")
	//fmt.Printf("Finished with Resopnse valie: %d", uint32(resp.ValidationStatus))
	//close(ctx.Done)

	return 0
	//return uint32(resp.ValidationStatus)
}

//export RunProxyVerify
func RunProxyVerify(data []byte, grpcClientID uint32) uint32 {

	log.Printf("++ [grpc client][RunProxy Verify] data: %v, clientID: %d\n", data, grpcClientID)
	job := NewJob(data, grpcClientID)
	log.Printf("++ [grpc client][RunProxy Verify] New job generated: %v\n", job)

	select {
	case jobChan <- *job:
		log.Printf("++ [grpc client][RunProxy Verify] Job was sent through Job channel (clientID:%d)\n", grpcClientID)
		return 0
		/*
			default:
				return 1
		*/
	}
}

func ProxyVerify(data []byte, grpcClientID uint32, jobDone chan bool, workerId int32) uint32 {

	log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] ProxyVerify Update Count : %d\n", workerId, g_count)
	if g_count == 0 {
		start = time.Now()
	}
	g_count++

	cli := client.cli
	log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] client data: %#v\n", workerId, client)

	if data == nil {
		data = []byte(defaultName)
	}
	log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] input data for Proxy Verify Stream: %#v\n", data)

	stream, err := cli.ProxyVerifyStream(context.Background(),
		&pb.ProxyVerifyRequest{
			Data:         data,
			Length:       uint32(len(data)),
			GrpcClientID: grpcClientID,
		},
	)
	if err != nil {
		log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] open stream error %v\n", err)
	}
	log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] stream info: %#v\n", workerId, stream)

	ctx := stream.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// XXX: Unfortunately, stream object is not thread-safe, so that it can't be used
	//		for multi-go routines, otherwise stream.Recv() receives an arbituary Send() call
	//		which was sent from the stream server
	go func(stream pb.SRxApi_ProxyVerifyStreamClient, jobDone chan bool, workerId int32) {
		defer close(jobDone)
		for {
			//fmt.Printf("[WorkerID: %d] (in go func) stream: %#v\n", workerId, stream)
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] EOF close \n", workerId)
				return
			}
			if err != nil {
				log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] Error - can not receive %v\n", workerId, err)
			}

			log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d](inside go routine, after stream Recv) stream: %#v\n",
				workerId, stream)
			log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] update count: %d, response Notify data : %#v\n",
				workerId, g_count, resp)
			log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] size : %#v\n", workerId, resp.Length)

			if resp.Type == 0 && resp.Length == 0 {
				_, _, line, _ := runtime.Caller(0)
				log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d](line:%d) close stream notify received \n",
					workerId, line+1)
				return
				//close(jobDone)
			} else {

				go_vn := &Go_ProxyVerifyNotify{
					_type:         uint8(resp.Type),
					_resultType:   uint8(resp.ResultType),
					_roaResult:    uint8(resp.RoaResult),
					_bgpsecResult: uint8(resp.BgpsecResult),
					_aspaResult:   uint8(resp.AspaResult),
					_length:       resp.Length,
					_requestToken: resp.RequestToken,
					_updateID:     resp.UpdateID,
				}
				vn := (*C.SRXPROXY_VERIFY_NOTIFICATION)(C.malloc(C.sizeof_SRXPROXY_VERIFY_NOTIFICATION))
				defer C.free(unsafe.Pointer(vn))
				go_vn.Pack(unsafe.Pointer(vn))
				//log.Printf("[WorkerID: %d] vn: %#v\n", workerId, vn)

				// to avoid runtime: address space conflict:
				//			and fatal error: runtime: address space conflict
				//	    NEED to make a shared library at the client side same way at server side
				C.processVerifyNotify_grpc(vn)

				// signal when the notification is over from SRx server
				// TODO: need to consider to have a flag that indicates ROA validation result was received
				if resp.Type == 0x06 && resp.ResultType == 0x02 && resp.RequestToken == 0x0 {
					return
					//close(jobDone)
				}
			}
		}
		log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] Stream Go routine Ended", workerId)
	}(stream, jobDone, workerId)

	go func() {
		//defer close(jobDone)
		<-ctx.Done()
		log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] Client Context Done (the Reason follows)", workerId)
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
	}()

	<-jobDone
	if g_count >= NUM_PREFIX {
		elapsed = time.Since(start)
		os.Stdout = g_std
		log.SetOutput(os.Stdout)
		log.Printf(" count: %d  took %s\n", g_count, elapsed)
		g_count = 0

		/* printout discard again */
		os.Stdout = nil
		log.SetOutput(ioutil.Discard)
	}

	log.Printf("+ [grpc client][ProxyVerify][WorkerID:%d] Finished with Resopnse value", workerId)
	return 0
}

func ImpleProxyVerifyBiStream() uint32 {

	cli := client.cli
	log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream] called \033[0m\n")

	stream, err := cli.ProxyVerifyBiStream(context.Background())
	if err != nil {
		log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream] open stream error %v \033[0m\n", err)
	}

	ctx := stream.Context()
	//ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	//defer cancel()
	var workerId int32
	Done := make(chan bool)

	go func() {
		for {
			log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream](SEND) WAITING next Verify request data ...\033[0m\n")
			select {
			case job := <-chVerifyData:
				workerId = job.workerId
				req := pb.ProxyVerifyRequest{
					Data:         job.data,
					Length:       uint32(len(job.data)),
					GrpcClientID: job.grpcClientID,
				}

				log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][workerID:%d](SEND) req data :%#v\033[0m\n",
					job.workerId, req)
				if err := stream.Send(&req); err != nil {
					log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][workerID:%d](SEND) send error %v\033[0m\n",
						job.workerId, err)

					if err := stream.CloseSend(); err != nil {
						log.Println(err)
					}
					return
				}
				log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][workerID:%d](SEND) %d bytes sent\033[0m",
					job.workerId, req.Length)
			}
		}

	}()
	// XXX: Unfortunately, stream object is not thread-safe, so that it can't be used
	//		for multi-go routines, otherwise stream.Recv() receives an arbituary Send() call
	//		which was sent from the stream server
	go func() {
		defer close(Done)
		for {
			//fmt.Printf("[WorkerID: %d] (in go func) stream: %#v\n", workerId, stream)
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream](RECV) EOF close \033[0m\n")
				return
			}
			if err != nil {
				log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream](RECV) Error - can not receive %v \033[0m\n",
					err)
				return
			}
			log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][WorkerID:%d](RECV) responses: %#v \033[0m\n",
				workerId, resp)
			// only if gw is not nil
			if gw != nil {
				fmt.Fprintf(gw, "\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][WorkerID:%d](RECV) responses: %#v \033[0m\n",
					workerId, resp)
			}

			go_vn := &Go_ProxyVerifyNotify{
				_type:         uint8(resp.Type),
				_resultType:   uint8(resp.ResultType),
				_roaResult:    uint8(resp.RoaResult),
				_bgpsecResult: uint8(resp.BgpsecResult),
				_aspaResult:   uint8(resp.AspaResult),
				_length:       resp.Length,
				_requestToken: resp.RequestToken,
				_updateID:     resp.UpdateID,
			}
			vn := (*C.SRXPROXY_VERIFY_NOTIFICATION)(C.malloc(C.sizeof_SRXPROXY_VERIFY_NOTIFICATION))
			defer C.free(unsafe.Pointer(vn))
			go_vn.Pack(unsafe.Pointer(vn))
			//log.Printf("[WorkerID: %d] vn: %#v\n", workerId, vn)

			// to avoid runtime: address space conflict:
			//			and fatal error: runtime: address space conflict
			//	    NEED to make a shared library at the client side same way at server side
			C.processVerifyNotify_grpc(vn)

			// signal when the notification is over from SRx server
			// TODO: need to consider to have a flag that indicates ROA validation result was received
			if resp.Type == 0x06 && resp.ResultType == 0x02 && resp.RequestToken == 0x0 {

				log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][WorkerID:%d] Something Wrong \033[0m \n",
					workerId)
				//return
			}
		}
		log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][WorkerID:%d] Stream Go routine Ended \033[0m \n",
			workerId)
	}()

	go func() {
		<-ctx.Done()
		log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][WorkerID:%d] Client Context Done (Reason follows) \033[0m \n", workerId)
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
	}()

	<-Done
	log.Printf("\033[1;33m+ [grpc client][ImpleProxyVerifyBiStream][WorkerID:%d] Finished with Resopnse value \033[0m \n",
		workerId)
	return 0
}

//export ImpleProxyDeleteUpdate
func ImpleProxyDeleteUpdate(data []byte, grpcClientID uint32) bool {

	cli := client.cli
	log.Printf("++ [grpc client][ImpleProxyDeleteUpdate] clientID: %0x input data : %#v \n", grpcClientID, data)

	req := pb.SerialPduDeleteUpdate{
		Data:         data,
		Length:       uint32(len(data)),
		GrpcClientID: grpcClientID,
	}

	resp, err := cli.ProxyDeleteUpdate(context.Background(), &req)

	if err != nil {
		log.Printf("could not receive: %v", err)
		return false
	}

	log.Printf(" response Status:%v, resp: %#v\n", resp.ValidationStatus, resp)
	return true
}

func hello(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found. --", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "get_post_form.html")
	case "POST":
		// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		fmt.Fprintf(w, "Post from website! r.PostFrom = %v\n", r.PostForm)
		srx_addr := r.FormValue("SRx address")
		verify := r.FormValue("Verify Request")
		ov := r.FormValue("Origin Validation")
		av := r.FormValue("ASPA Validation")

		fmt.Fprintf(w, "SRx Address = %s\n", srx_addr)
		fmt.Fprintf(w, "Verify = %s\n", verify)
		fmt.Fprintf(w, "Origin Valdiation = %s\n", ov)
		fmt.Fprintf(w, "ASPA Valdiation = %s\n", av)

		call_grpc_verify(w, r)
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func httpMain() {
	http.HandleFunc("/", hello)

	fmt.Printf("Starting server for testing HTTP POST...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

}

func call_grpc_verify(w http.ResponseWriter, r *http.Request) {

	gw = w

	log.Printf("main start Init(%s)\n", address)
	rv := InitSRxGrpc(address)
	if rv != true {
		log.Printf(" Init Error ")
		return
	}

	log.Printf("[main] Hello Request\n")
	buff_hello_request := []byte{0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x18, 0xa, 0x0, 0x32, 0x5,
		0x0, 0x0, 0xea, 0x61, 0x0, 0x0, 0x0, 0x0}

	res, grpcClientID := RunProxyHello(buff_hello_request)
	log.Printf("[main] Transferred: %#v, proxyID: %d\n\n", res, grpcClientID)
	fmt.Fprintf(w, "[main] Transferred: %#v, proxyID: %d\n\n", res, grpcClientID)

	// NOTE: SRx Proxy Verify
	log.Printf("[main] Verify Request\n")
	buff_verify_req_2 := []byte{0x3, 0x86, 0x1, 0x1, 0x1, 0x0, 0x2, 0x2, 0x0, 0x0, 0x1, 0x13, 0x3, 0x3,
		0x3, 0x18, 0x0, 0x0, 0x0, 0x3, 0xde, 0x1, 0x1, 0x0, 0x0, 0x0, 0xea, 0x63, 0x0, 0x0, 0x0, 0xd7,
		0x0, 0x2, 0x0, 0xcf, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xea, 0x61, 0x0, 0x0, 0xea, 0x62, 0x0, 0x0, 0xea, 0x63, 0x90,
		0x21, 0x0, 0xcb, 0x0, 0xe, 0x1, 0x0, 0x0, 0x0, 0xea, 0x62, 0x1, 0x0, 0x0, 0x0, 0xea, 0x63, 0x0,
		0xbd, 0x1, 0x45, 0xca, 0xd0, 0xac, 0x44, 0xf7, 0x7e, 0xfa, 0xa9, 0x46, 0x2, 0xe9, 0x98, 0x43,
		0x5, 0x21, 0x5b, 0xf4, 0x7d, 0xcd, 0x0, 0x48, 0x30, 0x46, 0x2, 0x21, 0x0, 0xcc, 0xc0, 0x4b, 0x6f,
		0xc2, 0x99, 0xcb, 0x8f, 0xbe, 0x1d, 0x69, 0x31, 0xb, 0x5c, 0x68, 0x5b, 0xc9, 0x47, 0x34, 0xc6,
		0xd5, 0xbb, 0xf, 0xe5, 0x7c, 0x8a, 0x43, 0x34, 0x4, 0x9e, 0x74, 0x85, 0x2, 0x21, 0x0, 0xf2, 0x16,
		0x4d, 0x64, 0x16, 0x61, 0x6b, 0xa3, 0xef, 0x41, 0x16, 0x41, 0xca, 0x8b, 0xa2, 0x7, 0xca, 0x45, 0xf9,
		0x8c, 0x91, 0xd5, 0x53, 0x62, 0x5d, 0x8c, 0xa0, 0x3a, 0x6, 0xcd, 0x12, 0x58, 0xc3, 0x4, 0x33, 0xfa,
		0x19, 0x75, 0xff, 0x19, 0x31, 0x81, 0x45, 0x8f, 0xb9, 0x2, 0xb5, 0x1, 0xea, 0x97, 0x89, 0xdc, 0x0,
		0x46, 0x30, 0x44, 0x2, 0x20, 0x3a, 0x94, 0xb6, 0xbf, 0xbb, 0x9e, 0xe, 0x9d, 0x65, 0x39, 0x13, 0x16,
		0x41, 0xf1, 0xc4, 0xd2, 0x5f, 0x7d, 0x4b, 0x37, 0x6a, 0xef, 0x5c, 0x50, 0x46, 0x32, 0x5c, 0x48, 0x16,
		0x6d, 0x13, 0x3, 0x2, 0x20, 0x61, 0xc9, 0xd6, 0x65, 0x8e, 0x83, 0xad, 0x49, 0x66, 0x30, 0xa1, 0x96,
		0x1d, 0xc6, 0x4e, 0x89, 0xc, 0x54, 0xe4, 0x7a, 0x27, 0x17, 0x67, 0xb1, 0x97, 0x99, 0x1b, 0x57, 0xea,
		0xc3, 0x16, 0x33}
	//RunProxyVerify(buff_verify_req, grpcClientID)
	log.Println("[main] Send Run Proxy Verify request ")
	RunProxyVerify(buff_verify_req_2, grpcClientID)

	// TODO : later, it needs to have channel variable to block/ unblock
	<-time.After(1 * time.Second)
	// NOTE: SRx PROY GOODBYE
	goGB := &Go_ProxyGoodBye{
		_type:       0x02,
		_keepWindow: binary.BigEndian.Uint16([]byte{0x83, 0x03}), // 0x03 0x84 : 900
		_reserved:   0x0,
		_zero:       0x0,
		_length:     binary.BigEndian.Uint32([]byte{0xc, 0x00, 0x00, 0x00}),
	}

	gb := (*C.SRXPROXY_GOODBYE)(C.malloc(C.sizeof_SRXPROXY_GOODBYE))
	defer C.free(unsafe.Pointer(gb))

	goGB.Pack(unsafe.Pointer(gb))
	log.Printf("[main]  goodbye message: %#v\n", gb)
	fmt.Fprintf(w, "[main]  goodbye message: %#v\n", gb)

	status := RunProxyGoodBye(*gb, uint32(grpcClientID))
	log.Printf("[main]  GoodBye response status: %#v\n", status)

	log.Println("[main] terminated")
}

func main() {
	///* FIXME XXX
	go httpMain()

	/*
		// Channel SEnd Recv priority Test
		// Conclusion: Recv should be prepared first and then send something to the channel, otherwise blocked
		//
		var wgTest sync.WaitGroup
		chTest := make(chan bool)
		wgTest.Add(1)
		go func() {
			wgTest.Done()
			fmt.Printf("----- receiving \n")
			<-chTest
			fmt.Printf("test goroutine done\n")
		}()

		wgTest.Wait()
		fmt.Printf("----- sending \n")
		chTest <- true
		fmt.Printf("test channel sending done\n")
	*/

	log.Printf("main start Init(%s)\n", address)
	rv := InitSRxGrpc(address)
	if rv != true {
		log.Printf(" Init Error ")
		return
	}
	//defer client.conn.Close()

	/*
		// TODO: construct Proxy Verify Request data structure and nested structures too
		req := pb.ProxyVerifyV4Request{}
		req.Common = &pb.ProxyBasicHeader{
			Type:         0x03,
			Flags:        0x83,
			RoaResSrc:    0x01,
			BgpsecResSrc: 0x01,
			Length:       0xa9,
			RoaDefRes:    0x03,
			BgpsecDefRes: 0x03,
			PrefixLen:    0x18,
			RequestToken: binary.BigEndian.Uint32([]byte{0x01, 0x00, 0x00, 0x00}),
		}
		req.PrefixAddress = &pb.IPv4Address{
			AddressOneof: &pb.IPv4Address_U8{
				U8: []byte{0x064, 0x01, 0x00, 0x00},
			},
		}
		req.OriginAS = binary.BigEndian.Uint32([]byte{0x00, 0x00, 0xfd, 0xf3})
		req.BgpsecLength = binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x00, 0x71})

		req.BgpsecValReqData = &pb.BGPSECValReqData{
			NumHops: binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x00, 0x01}),
			AttrLen: binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x00, 0x6d}),
			ValPrefix: &pb.SCA_Prefix{
				Afi:  binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x00, 0x6d}),
				Safi: binary.BigEndian.Uint32([]byte{0x00, 0x00, 0x00, 0x6d}),
			},
			ValData: &pb.BGPSEC_DATA_PTR{
				LocalAs: binary.BigEndian.Uint32([]byte{0x00, 0x00, 0xfd, 0xed}),
			},
		}
		fmt.Printf(" request: %#v\n", req)
		log.Fatalf("terminate here")
	*/

	// NOTE: SRx Proxy Hello
	log.Printf("[main] Hello Request\n")
	buff_hello_request := []byte{0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x18, 0xa, 0x0, 0x32, 0x5,
		0x0, 0x0, 0xea, 0x61, 0x0, 0x0, 0x0, 0x0}

	// NOTE: if this code goes to C. responseGRPC then it will have segmentation fault
	// so, don't go further
	res, grpcClientID := RunProxyHello(buff_hello_request)
	//r := Run(buff_hello_request)
	log.Printf("[main] Transferred: %#v, proxyID: %d\n\n", res, grpcClientID)
	//*/

	log.Println("---------------")
	// NOTE: SRx Proxy GoodBye Stream Test
	/*
		go func() {
			log.Printf("[main] GoodBye Stream Request\n")
			buff_goodbye_stream_request := []byte{0x02, 0x03, 0x84, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0c}
			<-time.After(2 * time.Second)
			result := RunProxyGoodByeStream(buff_goodbye_stream_request, grpcClientID)
			log.Println("[main] result:", result)
		}()
	*/

	/*
		go func() {
			<-time.After(2 * time.Second)
			log.Printf("[main] Sync Request Stream \n")
			buff_sync_request_stream_request := []byte{0x0a, 0x00, 0x00, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0c}
			result := RunProxyStream(buff_sync_request_stream_request, grpcClientID)
			log.Println("[main] result:", result)
		}()
	*/

	//time.Sleep(2 * time.Second)

	// NOTE: SRx Proxy Verify
	log.Printf("[main] Verify Request\n")
	/*
		buff_verify_req := []byte{0x03, 0x83, 0x01, 0x01, 0x00, 0x00, 0x00, 0xa9, 0x03, 0x03, 0x00, 0x18,
			0x00, 0x00, 0x00, 0x01, 0x64, 0x01, 0x00, 0x00, 0x00, 0x00, 0xfd, 0xf3, 0x00, 0x00, 0x00, 0x71,
			0x00, 0x01, 0x00, 0x6d, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfd, 0xed, 0x00, 0x00, 0xfd, 0xf3,
			0x90, 0x21, 0x00, 0x69, 0x00, 0x08, 0x01, 0x00, 0x00, 0x00, 0xfd, 0xf3, 0x00, 0x61, 0x01, 0xc3,
			0x04, 0x33, 0xfa, 0x19, 0x75, 0xff, 0x19, 0x31, 0x81, 0x45, 0x8f, 0xb9, 0x02, 0xb5, 0x01, 0xea,
			0x97, 0x89, 0xdc, 0x00, 0x48, 0x30, 0x46, 0x02, 0x21, 0x00, 0xbd, 0x92, 0x9e, 0x69, 0x35, 0x6e,
			0x7b, 0x6c, 0xfe, 0x1c, 0xbc, 0x3c, 0xbd, 0x1c, 0x4a, 0x63, 0x8d, 0x64, 0x5f, 0xa0, 0xb7, 0x20,
			0x7e, 0xf3, 0x2c, 0xcc, 0x4b, 0x3f, 0xd6, 0x1b, 0x5f, 0x46, 0x02, 0x21, 0x00, 0xb6, 0x0a, 0x7c,
			0x82, 0x7f, 0x50, 0xe6, 0x5a, 0x5b, 0xd7, 0x8c, 0xd1, 0x81, 0x3d, 0xbc, 0xca, 0xa8, 0x2d, 0x27,
			0x47, 0x60, 0x25, 0xe0, 0x8c, 0xda, 0x49, 0xf9, 0x1e, 0x22, 0xd8, 0xc0, 0x8e}
	*/

	buff_verify_req_2 := []byte{0x3, 0x86, 0x1, 0x1, 0x1, 0x0, 0x2, 0x2, 0x0, 0x0, 0x1, 0x13, 0x3, 0x3,
		0x3, 0x18, 0x0, 0x0, 0x0, 0x3, 0xde, 0x1, 0x1, 0x0, 0x0, 0x0, 0xea, 0x63, 0x0, 0x0, 0x0, 0xd7,
		0x0, 0x2, 0x0, 0xcf, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xea, 0x61, 0x0, 0x0, 0xea, 0x62, 0x0, 0x0, 0xea, 0x63, 0x90,
		0x21, 0x0, 0xcb, 0x0, 0xe, 0x1, 0x0, 0x0, 0x0, 0xea, 0x62, 0x1, 0x0, 0x0, 0x0, 0xea, 0x63, 0x0,
		0xbd, 0x1, 0x45, 0xca, 0xd0, 0xac, 0x44, 0xf7, 0x7e, 0xfa, 0xa9, 0x46, 0x2, 0xe9, 0x98, 0x43,
		0x5, 0x21, 0x5b, 0xf4, 0x7d, 0xcd, 0x0, 0x48, 0x30, 0x46, 0x2, 0x21, 0x0, 0xcc, 0xc0, 0x4b, 0x6f,
		0xc2, 0x99, 0xcb, 0x8f, 0xbe, 0x1d, 0x69, 0x31, 0xb, 0x5c, 0x68, 0x5b, 0xc9, 0x47, 0x34, 0xc6,
		0xd5, 0xbb, 0xf, 0xe5, 0x7c, 0x8a, 0x43, 0x34, 0x4, 0x9e, 0x74, 0x85, 0x2, 0x21, 0x0, 0xf2, 0x16,
		0x4d, 0x64, 0x16, 0x61, 0x6b, 0xa3, 0xef, 0x41, 0x16, 0x41, 0xca, 0x8b, 0xa2, 0x7, 0xca, 0x45, 0xf9,
		0x8c, 0x91, 0xd5, 0x53, 0x62, 0x5d, 0x8c, 0xa0, 0x3a, 0x6, 0xcd, 0x12, 0x58, 0xc3, 0x4, 0x33, 0xfa,
		0x19, 0x75, 0xff, 0x19, 0x31, 0x81, 0x45, 0x8f, 0xb9, 0x2, 0xb5, 0x1, 0xea, 0x97, 0x89, 0xdc, 0x0,
		0x46, 0x30, 0x44, 0x2, 0x20, 0x3a, 0x94, 0xb6, 0xbf, 0xbb, 0x9e, 0xe, 0x9d, 0x65, 0x39, 0x13, 0x16,
		0x41, 0xf1, 0xc4, 0xd2, 0x5f, 0x7d, 0x4b, 0x37, 0x6a, 0xef, 0x5c, 0x50, 0x46, 0x32, 0x5c, 0x48, 0x16,
		0x6d, 0x13, 0x3, 0x2, 0x20, 0x61, 0xc9, 0xd6, 0x65, 0x8e, 0x83, 0xad, 0x49, 0x66, 0x30, 0xa1, 0x96,
		0x1d, 0xc6, 0x4e, 0x89, 0xc, 0x54, 0xe4, 0x7a, 0x27, 0x17, 0x67, 0xb1, 0x97, 0x99, 0x1b, 0x57, 0xea,
		0xc3, 0x16, 0x33}
	//RunProxyVerify(buff_verify_req, grpcClientID)
	log.Println("[main] Send Run Proxy Verify request ")
	RunProxyVerify(buff_verify_req_2, grpcClientID)
	//RunStream(buff_verify_req)

	// TODO : later, it needs to have channel variable to block/ unblock
	<-time.After(5 * time.Second)
	// NOTE: SRx PROY GOODBYE
	goGB := &Go_ProxyGoodBye{
		_type:       0x02,
		_keepWindow: binary.BigEndian.Uint16([]byte{0x83, 0x03}), // 0x03 0x84 : 900
		_reserved:   0x0,
		_zero:       0x0,
		_length:     binary.BigEndian.Uint32([]byte{0xc, 0x00, 0x00, 0x00}),
	}

	gb := (*C.SRXPROXY_GOODBYE)(C.malloc(C.sizeof_SRXPROXY_GOODBYE))
	defer C.free(unsafe.Pointer(gb))

	goGB.Pack(unsafe.Pointer(gb))
	log.Printf("[main]  goodbye message: %#v\n", gb)

	status := RunProxyGoodBye(*gb, uint32(grpcClientID))
	log.Printf("[main]  GoodBye response status: %#v\n", status)

	log.Println("[main] terminated")
	/* FIXME
	data := []byte(defaultName)
	data2 := []byte{0x10, 0x11, 0x40, 0x42}
	data3 := []byte{0x10, 0x11, 0x40, 0x42, 0xAB, 0xCD, 0xEF}

	r := Run(data)
	log.Printf("Transferred: %#v\n\n", r)

	r = Run(data2)
	log.Printf("Transferred: %#v\n\n", r)

	r = RunStream(data3)
	log.Printf("Transferred: %#v\n\n", r)
	*/
}

/* NOTE

TODO 1: init function - for receiving client (*grpc.ClientConn)
		--> maybe good to use a global variable for client

TODO 2

*/
