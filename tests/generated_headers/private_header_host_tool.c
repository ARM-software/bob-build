#include "h1.h"
#include <stdio.h>

int main(int argc, char **argv)
{
    if (argc != 2)
        return 1;
    FILE *output = fopen(argv[1], "w");
    if (!output)
        return 1;
    if (fprintf(output, "#define HOST_TOOL_RESULT %d\n", H1) < 0)
        return 1;
    return fclose(output) != 0;
}
