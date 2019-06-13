#include "external_shared.h"

int external_shared_via_proxy(void) {
    return external_shared();
}
