
#include <stdio.h>
#include "libsrx_grpc_client.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>

static void* GoodByeStreamThread(void *arg);

int main ()
{
    printf("[%s] Running C imple grpc client from C\n", __FILE__);
    int32_t result;

#if 0
    /* test 1 */
    char buff[10];
    buff[0] = 0xAB;
    buff[1] = 0xCD;
    buff[2] = 0xEF;
    GoSlice pdu = {(void*)buff, (GoInt)3, (GoInt)10};
    result = Run(pdu);
#endif



    /* test 2: request HelloRequest */
    /* 
      hdr->type            = PDU_SRXPROXY_HELLO;
      hdr->version         = htons(SRX_PROTOCOL_VER);
      hdr->length          = htonl(length);
      hdr->proxyIdentifier = htonl(5);   // htonl(proxy->proxyID);
      hdr->asn             = htonl(65005); 
      hdr->noPeers         = htonl(noPeers); 
    */

    // XXX init process
    //
    GoString gs_addr = {
        p : "localhost:50000",
        n : 15
    };
    gs_addr.n = strlen((const char*)gs_addr.p);
    printf("size: %d \n", gs_addr.n);

    bool res = InitSRxGrpc(gs_addr);
    printf("init result: %d \n", res);


    // XXX Hello Request
    char buff_hello_request[20] = 
    {0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x14, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0xfd, 0xed, 0x0, 0x0, 0x0, 0x0};

    GoSlice hello_requiest_pdu = {(void*)buff_hello_request, (GoInt)20, (GoInt)20};
    //result = Run(hello_requiest_pdu);
    struct RunProxyHello_return resp = RunProxyHello(hello_requiest_pdu);
    printf("Hello Request Result ID: %02x\n", (uint8_t)resp.r1);

    unsigned int grpcClientID = resp.r1;


    pthread_t tid;
    pthread_attr_t attr;
    pthread_attr_init(&attr);
    pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
    printf("+ pthread grpc service started...\n");

    int ret = pthread_create(&tid, &attr, GoodByeStreamThread, (void*)&grpcClientID);
    if (ret != 0)
    {
        RAISE_ERROR("Failed to create a grpc thread");
    }
    printf("+ pthread created \n");
    pthread_join(tid, NULL);


    // XXX Verify 
    //
    /* test 3: request verification */
    printf("[%s] Validation  Request\n", __FILE__ );
    printf("[%s] verify update sent\n", __FILE__);
    /* verify update srxproxy protocol test */
    char verify_buff[169] = {
        0x03, 0x83, 0x01, 0x01, 0x00, 0x00, 0x00, 0xa9, 0x03, 0x03, 0x00, 0x18,
        0x00, 0x00, 0x00, 0x01, 0x64, 0x01, 0x00, 0x00, 0x00, 0x00, 0xfd, 0xf3, 0x00, 0x00, 0x00, 0x71,
        0x00, 0x01, 0x00, 0x6d, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfd, 0xed, 0x00, 0x00, 0xfd, 0xf3,
        0x90, 0x21, 0x00, 0x69, 0x00, 0x08, 0x01, 0x00, 0x00, 0x00, 0xfd, 0xf3, 0x00, 0x61, 0x01, 0xc3,
        0x04, 0x33, 0xfa, 0x19, 0x75, 0xff, 0x19, 0x31, 0x81, 0x45, 0x8f, 0xb9, 0x02, 0xb5, 0x01, 0xea,
        0x97, 0x89, 0xdc, 0x00, 0x48, 0x30, 0x46, 0x02, 0x21, 0x00, 0xbd, 0x92, 0x9e, 0x69, 0x35, 0x6e,
        0x7b, 0x6c, 0xfe, 0x1c, 0xbc, 0x3c, 0xbd, 0x1c, 0x4a, 0x63, 0x8d, 0x64, 0x5f, 0xa0, 0xb7, 0x20,
        0x7e, 0xf3, 0x2c, 0xcc, 0x4b, 0x3f, 0xd6, 0x1b, 0x5f, 0x46, 0x02, 0x21, 0x00, 0xb6, 0x0a, 0x7c,
        0x82, 0x7f, 0x50, 0xe6, 0x5a, 0x5b, 0xd7, 0x8c, 0xd1, 0x81, 0x3d, 0xbc, 0xca, 0xa8, 0x2d, 0x27,
        0x47, 0x60, 0x25, 0xe0, 0x8c, 0xda, 0x49, 0xf9, 0x1e, 0x22, 0xd8, 0xc0, 0x8e
    };
    
    GoSlice verify_pdu = {(void*)verify_buff, (GoInt)169, (GoInt)170};
    //result = RunStream(verify_pdu);
    result = RunProxyVerify(verify_pdu, grpcClientID);
    printf("[%s] Validation Result: %02x\n", __FILE__, result);


    // XXX GoodBye protocol
    // 
    uint32_t length = sizeof(SRXPROXY_GOODBYE);     
    uint8_t pdu[length];                            
    SRXPROXY_GOODBYE* hdr = (SRXPROXY_GOODBYE*)pdu; 
    memset(pdu, 0, length);                         

    printf("\n\nsend Goodbye! called\n\n"); 
    hdr->type       = PDU_SRXPROXY_GOODBYE;        
    hdr->keepWindow = htons(SRX_DEFAULT_KEEP_WINDOW);           
    hdr->length     = htonl(length);               

    RunProxyGoodBye(*hdr, grpcClientID);


    printf("main program terminated\n");
    return 0;
}

static void* GoodByeStreamThread(void *arg)
{
    // XXX TEST: GoodBye Stream
    // TODO: this needs to generate pthread and run concurrently
    //
    
    unsigned int grpcClientID = *((unsigned int*)arg);
    printf("Run Proxy Good Bye Stream \n");
    char buff_goodbye_stream_request[8] = {0x02, 0x03, 0x84, 0x0, 0x0, 0x0, 0x0, 0x08};
    GoSlice goodbye_stream_pdu = {(void*)buff_goodbye_stream_request, (GoInt)8, (GoInt)8};
    int result = RunProxyGoodByeStream (goodbye_stream_pdu, grpcClientID);
    
    printf("Run Proxy Good Bye Stream terminated \n");

}

#if 0

typedef struct {
  uint8_t       type;          // 3 and 4
  uint8_t       flags;
  uint8_t       roaResSrc;
  uint8_t       bgpsecResSrc;
  uint32_t      length;
  uint8_t       roaDefRes;
  uint8_t       bgpsecDefRes;
  uint8_t       zero;
  uint8_t       prefixLen;
  uint32_t      requestToken; // Added with protocol version 1.0
} __attribute__((packed)) SRXRPOXY_BasicHeader_VerifyRequest;


typedef struct {
  SRXRPOXY_BasicHeader_VerifyRequest common; // type = 3
  IPv4Address      prefixAddress;
  uint32_t         originAS;
  uint32_t         bgpsecLength;
  BGPSECValReqData bgpsecValReqData;
} __attribute__((packed)) SRXPROXY_VERIFY_V4_REQUEST;

typedef union {
  struct in_addr in_addr;
  uint32_t       u32;
  uint8_t        u8[4];
} IPv4Address;

typedef struct {
  uint16_t   numHops;
  uint16_t   attrLen;
  SCA_Prefix valPrefix;
  BGPSEC_DATA_PTR valData;
} __attribute__((packed)) BGPSECValReqData;


typedef struct                                                    
{                                                                 
  u_int16_t afi;                                                  
  u_int8_t  safi;                                                 
  u_int8_t  length;                                               
  union                                                           
  {                                                               
    struct in_addr  ipV4;                                         
    struct in6_addr ipV6;                                         
    u_int8_t ip[16];                                              
  } addr;                                                         
} __attribute__((packed)) SCA_Prefix;                             


typedef struct {
  uint32_t  local_as;
} BGPSEC_DATA_PTR;

#endif
