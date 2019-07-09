#include <stdio.h>

#if BIN_FLAG != 1 || defined(LIB_FLAG)
    #error "Incorrect cflags in library build"
#endif

int helper(void);

int main(void)
{
    printf("Hello, world!\n");
    return helper();
}
