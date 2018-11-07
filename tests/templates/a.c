#ifndef TEMPLATE_TEST_VALUE
#error TEMPLATE_TEST_VALUE is not defined !
#endif

#if TEMPLATE_TEST_VALUE_HOST != TEMPLATE_TEST_VALUE && TEMPLATE_TEST_VALUE_TARGET != TEMPLATE_TEST_VALUE
#error Neither TEMPLATE_TEST_VALUE HOST or TARGET correctly defined !
#endif

int main() {
    return 0;
}
