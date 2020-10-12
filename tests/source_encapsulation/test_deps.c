#ifdef __has_include
#if __has_include("must_not_have.h")
    #error "must_not_have.h should not be available!"
#endif
#else
    #warning "Compiler does not support __has_include, unable to check header visibility"
#endif /* __has_include */

#include "must_have.h"
#include "types.h"

int main(void) {
    return D1 + D2 == 3;
}
