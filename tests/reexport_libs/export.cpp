#include <hidden.h>

static void checkForHiddenAvailibility()
{
	hiddenFunction();
}

void silenceUnusedFunctionError()
{
	checkForHiddenAvailibility();
}
