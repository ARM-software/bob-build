#include "tests/bazel_cc_import/bazel/library/simple/api.h"

int main(void) {
    return value_a() + value_b() == VALUE_SUM ? 0 : 1;
}
