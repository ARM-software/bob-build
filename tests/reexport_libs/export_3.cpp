#include <hidden.h>

static void checkForHiddenAvailibility_3()
{
	hiddenFunction();
}

void silenceUnusedFunctionError_3()
{
	checkForHiddenAvailibility_3();
}
