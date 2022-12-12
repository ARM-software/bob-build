#include "libs/lib.h"


int main()
{
    #ifdef FORWARDED_DEFINE
        return magic_value();
    #else
        #error "no forward define"
    #endif
}
