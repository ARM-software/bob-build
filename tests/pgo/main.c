#include <stdio.h>

// The PGO_USED macro will be set to 1 when building with e.g.
// `mm ANDROID_PGO_INSTRUMENT=pgo_test_benchmark`. Do not assert it here though,
// as in general the Bob tests are just run with `mm`.

int main(void) {
    printf("Hello PGO!\n");
    return 0;
}
