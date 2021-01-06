#include <stdio.h>

const char* output_hash(void);
int getValue(void);

int main(void) {

    printf("%s%d\n", output_hash(), getValue());

    return 0;
}
