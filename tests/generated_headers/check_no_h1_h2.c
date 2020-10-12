#ifdef __has_include
#if __has_include("h1.h") || __has_include("h2.h")
    #error "h1.1 and h2.h incorrectly exported via generated_headers"
#endif
#else
    #warning "Compiler does not support __has_include, unable to check header visibility"
#endif /* __has_include */

int main(void)
{
    return 0;
}
