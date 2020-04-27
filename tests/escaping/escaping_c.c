#include <assert.h>
#include <string.h>
#include <stdio.h>

/* Verify the macros are string literals */
static const char str[] = "" STRING;
static const char cmd[] = "" COMMAND;
static const char str1[] = "" STRING1;

int checkc(void) {
	int result = 0;

	printf("C STRING is `%s`\n", str);
	printf("C COMMAND is `%s`\n", cmd);
	printf("C STRING1 is `%s`\n", str1);

	/* Run time check macro values. Ideally we would do this at compile time,
	 * but this is difficult to get right across different compilers and
	 * versions. We rely on the C++ test to verify that the escaping is
	 * correct at compile time. */
	if (strcmp(str, "string") != 0) {
		printf("C STRING is incorrect\n");
		result = 1;
	}
	if (strcmp(cmd, "PATH=$PATH `uname` | true < /dev/random > /dev/null &") != 0) {
		printf("C COMMAND is incorrect\n");
		result = 1;
	}
	if (strcmp(str1, "string1") != 0) {
		printf("C STRING1 is incorrect\n");
		result = 1;
	}

	return result;
}
