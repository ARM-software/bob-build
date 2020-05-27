#include "libblah.h"
#include "libblah_feature.h"

int output(void) {
	return 42;
}

int feature_function(void) {
	return output() * 2;
}
