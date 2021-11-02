#include <stdio.h>
#include "libsrx_grpc_server_callback.h"


int mainCallback(int f);
int main()
{

    printf("using srxapi_server library from C\n");


    mainCallback(5);
    Serve();

    printf("Main program terminated\n");

}


int mainCallback(int f)
{
    printf("[%s] called with arg: %d \n", __FUNCTION__, f);
    cb_proxy(f);

    return 0;
}
