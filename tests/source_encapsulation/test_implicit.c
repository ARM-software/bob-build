#if __has_include("message.h")
    #define HAS_MSG 1
#endif

#include "types.h"
#include "impl/implicit.h"

int main(void) {
    return IMPL == D3 + HAS_MSG;
}
