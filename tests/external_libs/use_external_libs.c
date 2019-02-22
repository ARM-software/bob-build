#include "external_static.h"
#include "external_shared.h"
#include "external_header.h"

int main(void) {
    return external_static() + external_shared() + EXTERNAL_HEADER;
}
