#include "api.h"

int main(void) {
    if (value_a() + value_b() != VALUE_SUM) {
        return 1;
    }
    return 0;
}
