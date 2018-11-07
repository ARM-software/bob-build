#ifndef BOB_TEST_CFLAGS1_H
#define BOB_TEST_CFLAGS1_H

#ifndef FOO
#error FOO is not defined here !
#endif

#if FOO != 2
#error FOO is incorrectly defined here !
#endif

int bob_test_cflags1_test();

#endif
