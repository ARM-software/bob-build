#include <iostream>

extern "C" int checkc(void);

/* This function checks that the input strings are equivalent,
 * and can be called from static_assert */
constexpr bool equals(const char *s0, const char *s1) {

	/* In C++11 (and older compilers) we're only allowed a single
	 * return statement, so do this check using recursion. */
	return ((*s0 == *s1) &&
	        ((*s0 == '\0') ||
	         equals(s0+1, s1+1)));
}

/* Verify the macros are string literals */
static const char str[] = "" STRING;
static const char cmd[] = "" COMMAND;
static const char str2[] = "" STRING2;

/* Check macro values at compile time */
static_assert(equals(STRING, "string"), "bad macro definition STRING");
static_assert(equals(COMMAND, "PATH=$PATH `uname` | true < /dev/random > /dev/null &"),
              "bad macro definition COMMAND");
static_assert(equals(STRING2, "string2"), "bad macro definition STRING2");


int main(void) {
	// Make use of the static variables
	std::cout << "C++ STRING is `" << str << "`\n";
	std::cout << "C++ Command is `" << cmd << "`\n";
	std::cout << "C++ STRING2 is `" << str2 << "`\n";

	/* Link the C checks */
	return checkc();
}
