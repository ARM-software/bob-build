#include <stdio.h>

extern int do_c(int);
extern int do_d(int);

int main(int argc, const char **argv)
{
	int result = do_c(50) + do_d(40);
	printf("%d\n", result);
    return 0;
}
