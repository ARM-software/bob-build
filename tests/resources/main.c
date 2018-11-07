#include "b.h"

#if FOO != 1
    #error "FOO not propagated from sl_liba"
#endif

int main(void)
{
    (void) do_b(50);
    return 0;
}
