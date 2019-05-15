int func1(void);
int func2(void);

int main(void) {
	return (func1() == 4 && func2() == 7) ? 0 : 1;
}
