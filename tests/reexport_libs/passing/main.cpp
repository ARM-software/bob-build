void functionA();
void functionB();
void functionC();
void functionD();
void functionE();

int main()
{
	functionA();
	functionB();
	functionC();
	functionD();
	functionE();
}

#ifndef HAVE_A
#error "Should have A"
#endif

#ifndef HAVE_B
#error "Should have B"
#endif

#ifndef HAVE_D
#error "Should have D"
#endif

//////////////////////////////////////

#ifdef HAVE_C
#error "Shouldn't have C"
#endif

#ifdef HAVE_E
#error "Shouldn't have E"
#endif
