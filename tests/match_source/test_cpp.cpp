extern "C" {

int test_cxxflags()
{
#ifdef CXXFLAGS_TEST
    return 0;
#else
    return 1;
#endif
}

}
