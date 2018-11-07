#include "bob_test_cflags1.h"

#ifndef FOO
#error "FOO Should be definied"
#endif

#ifndef BAR
#error "BAR Should be definied"
#endif

int main() {
    return  -FOO + BAR;
}
