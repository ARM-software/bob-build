#include <zlib.h>

int main(void) {
	z_stream strm = { 0 };

	return deflateInit(&strm, Z_DEFAULT_COMPRESSION);
}
