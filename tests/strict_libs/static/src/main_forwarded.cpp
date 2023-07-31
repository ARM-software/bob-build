

int main()
{
    #ifdef FORWARDED_DEFINE
    // defines are forwarded, but not implementation
        return 0;
    #else
        #error "no forward define"
    #endif
}
