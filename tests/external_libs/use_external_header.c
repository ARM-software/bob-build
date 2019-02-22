#include "external_header.h"

int use_external_header(void) {
    return EXTERNAL_HEADER;
}

#if DEFINE_MAIN
int main(void) {
    return use_external_header();
}
#endif
