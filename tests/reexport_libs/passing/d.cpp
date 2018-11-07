void functionD()
{
}

#ifndef HAVE_D
#error "Should have D"
#endif

/////////////////////////////

#ifdef HAVE_A
#error "Shouldn't have A"
#endif

#ifdef HAVE_B
#error "Shouldn't have B"
#endif

#ifdef HAVE_C
#error "Shouldn't have C"
#endif

#ifdef HAVE_E
#error "Shouldn't have E"
#endif
