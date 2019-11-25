#include <hidden.h>

static void checkForHiddenAvailibility_2()
{
	hiddenFunction();
}

void silenceUnusedFunctionError_2()
{
	checkForHiddenAvailibility_2();
}
