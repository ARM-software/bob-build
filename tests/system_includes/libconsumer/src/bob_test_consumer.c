#include <bob_test_public_header.h>

int main() {
    /* Does not issue a warning when calling system provided API */
    const int test = bob_test_public_api_with_warning(0.0);
    (void)test;
    return  bob_test_public_api() == 42;
}
