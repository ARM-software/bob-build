//#include <linux/module.h>
#include "kernel_header.h"
#include "include/test_module1.h"

int test_int1 = 123;

int test_function1(void)
{
    return KERNEL_THING;
}

//EXPORT_SYMBOL(test_int1);
//EXPORT_SYMBOL(test_function1);
