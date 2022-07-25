
#include "a/a.h"
#include "b/b.h"

#ifndef A_VAL
#error "A_VAL not defined."
#endif

#ifndef B_VAL
#error "B_VAL not defined."
#endif

int main(void)
{
    return A_VAL + B_VAL;
}
