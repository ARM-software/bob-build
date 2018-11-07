#include "bob_test_liba.h"
#include "bob_test_liba2.h"

int bob_test_liba_test() {
    if (BOB_TEST_LIBA_MAGIC != BOB_TEST_LIBA_MAGIC)
        return -1;
    return BOB_TEST_LIBA_TEST_VALUE;
}
