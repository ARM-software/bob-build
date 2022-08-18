#include <stdio.h>
#include <stdlib.h>

void useless(void)
{
    printf("I am not used\n");
}

int main(int argc, char** argv)
{
    (void)argc;
    (void)argv;
    const int i = 0;
    return i;
}
