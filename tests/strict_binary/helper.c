#if LIB_FLAG != 1 || defined(BIN_FLAG)
    #error "Incorrect cflags in library build"
#endif

int helper(void)
{
    return 0;
}
