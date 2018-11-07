void functionC()
{
}

#ifndef HAVE_C
#error "Should have C"
#endif

#ifndef HAVE_E
#error "Should have E"
#endif

/////////////////////////////

#ifdef HAVE_A
#error "Shouldn't have A"
#endif

#ifdef HAVE_B
#error "Shouldn't have B"
#endif

#ifdef HAVE_D
#error "Shouldn't have D"
#endif
