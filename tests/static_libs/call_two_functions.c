int CALL1(int a);
int CALL2(int a);

int FUNCTION(int a)
{
	return CALL1(a) + CALL2(a) + 2;
}
