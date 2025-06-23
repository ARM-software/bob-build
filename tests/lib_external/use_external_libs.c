#include "external_static.h"
#include "external_shared.h"

int main(void) {
    return external_static() + external_shared();
}
