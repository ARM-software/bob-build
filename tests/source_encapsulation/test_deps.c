#if __has_include("must_not_have.h")
    #error "must_not_have.h should not be available!"
#endif

#include "must_have.h"
#include "types.h"

int main(void) {
    return D1 + D2 == 3;
}
