#include "libst/libst.h"
#include "libsh/libsh.h"

int main(void) {
    return libshared() + libstatic() == 42 ? 0 : 1;
}
