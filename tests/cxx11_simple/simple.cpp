#include <iostream>
#include <string>

#ifdef EXTRA_CONLYFLAGS
    #error "C-specific flags was set"
#endif
#ifndef EXTRA_CFLAGS
    #error "Common C flags were not set"
#endif
#ifndef EXTRA_CXXFLAGS
    #error "C++-specific flags were not set"
#endif

using namespace std;

int main() {
    cout << string("hello world!") << endl;
    return 0;
}
