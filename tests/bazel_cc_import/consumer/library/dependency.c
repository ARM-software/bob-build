#include "tests/bazel_cc_import/bazel/library/dependency/api.h"

int main(void) {
    return value_a() == VALUE_SUM ? 0 : 1;
}
