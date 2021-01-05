// Expect to have the exported_function() definition
// prepended to this file to test {{match_srcs}}

extern int another_function(void);
extern int test_cxxflags();

int main(void)
{
	int result = 0;

	{
		int tmp = exported_function();
		tmp += another_function();
		if (tmp != 46)
			result = 1;
	}

#ifndef CFLAGS_TEST
	result = 1;
#endif

#ifndef CONLYFLAGS_TEST
	result = 1;
#endif

	result = test_cxxflags();

	return result;
}
