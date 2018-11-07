void functionB()
{
}

#ifndef HAVE_B
#error "Should have B"
#endif

#ifndef HAVE_D
#error "Should have D"
#endif

/////////////////////////////

#ifdef HAVE_A
#error "Shouldn't have A"
#endif

#ifdef HAVE_C
#error "Shouldn't have C"
#endif

#ifdef HAVE_E
#error "Shouldn't have E"
#endif
