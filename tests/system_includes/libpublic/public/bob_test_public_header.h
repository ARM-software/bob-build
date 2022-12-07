#pragma once

#include <stdlib.h>

#define BOB_TEST_PUBLIC_DEF 1337

static inline int bob_test_public_api_with_warning(double x) {
	// double -> int, warns with -Wconversion
	return abs(x);
}

__attribute__ ((visibility ("default")))
int bob_test_public_api();
