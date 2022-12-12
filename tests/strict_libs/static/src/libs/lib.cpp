#include "lib.h"

int magic_value()
{
    #ifdef LOCAL_DEFINE
        return 0;
    #else
        #error "Local define missing from target rule."
    #endif
}
