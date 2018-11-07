#include <hidden.h>

static void checkForHiddenAvability()
{
	hiddenFunction();
}

void silenceUnusedFunctionError()
{
    checkForHiddenAvability();
}
