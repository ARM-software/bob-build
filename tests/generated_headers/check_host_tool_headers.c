#include "host_tool_output.h"
#include "nested_host_tool_output.h"

#if __has_include("h1.h")
#error "A host tool leaked its private build headers to a generated-header consumer"
#endif

#if HOST_TOOL_RESULT != 1
#error "The host tool did not use its own generated build header"
#endif

int main(void)
{
    return 0;
}
