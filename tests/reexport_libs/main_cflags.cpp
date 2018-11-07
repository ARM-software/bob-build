#ifdef SHOW_HIDDEN
// Expose hidden function
#include <hidden.h>
#endif

#include <cstdio>

int main()
{
	hiddenFunction();

#if defined ME_TARGET && defined ME_HOST
#error "Should be only 1"
#endif

#ifdef ME_TARGET
	printf("Target\n");
#endif
#ifdef ME_HOST
	printf("Host\n");
#endif

	printf("Yeap\n");

	return 0;
}
