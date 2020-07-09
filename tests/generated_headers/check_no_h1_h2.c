#if __has_include("h1.h") || __has_include("h2.h")
    #error "h1.1 and h2.h incorrectly exported via generated_headers"
#endif

int main(void)
{
    return 0;
}
