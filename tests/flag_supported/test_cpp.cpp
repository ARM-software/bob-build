#include <iostream>
#include <string>

using namespace std;

#ifdef EXTRA_CXXFLAGS
#define QUALIFIER    const
#else
#define QUALIFIER
#endif

QUALIFIER int hello() {
    cout << string("Hello World!") << endl;
    return 0;
}
