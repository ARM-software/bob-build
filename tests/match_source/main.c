// Expect to have the exported_function() definition
// prepended to this file to test {{match_srcs}}

extern int another_function(void);

int main(void)
{
	int result = exported_function();
	result += another_function();
	return (result == 46) ? 0 : 1;
}
