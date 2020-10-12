#include "1.h"
#include "2.h"
#include "3.h"

#ifdef __has_include
#if __has_include("must_not_have.h")
    #error "must_not_have.h should not be available!"
#endif
#else
    #warning "Compiler does not support __has_include, unable to check header visibility"
#endif /* __has_include */

#if D1 != 1 || D2 != 2 || D3 != 3
    #error "Incorrect values of D1, D2 or D3"
#endif

int main(void) {
    return 0;
}
