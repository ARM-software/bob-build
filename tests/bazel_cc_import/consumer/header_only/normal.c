#include "tests/bazel_cc_import/bazel/header_only/normal/api.h"

int main(void) {
    return value_a() + value_b() == VALUE_SUM ? 0 : 1;
}
