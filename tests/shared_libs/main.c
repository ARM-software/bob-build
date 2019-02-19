#include <stdio.h>
#include <string.h>

int sharedtest_installed(void);
int sharedtest_not_installed(void);

int main(int argc, char **argv) {
    /* To verify that this works for host libraries too, and that the test is
     * run on Android, use the host version of this to generate a source file
     * for another target binary. */
    if (argc > 1) {
        FILE *fp = fopen(argv[1], "wt");
        fprintf(fp, "int main(void) { return 0; }\n");
        fclose(fp);
    }

    if (sharedtest_installed() == 12345 && sharedtest_not_installed() == 12345) {
        return 0;
    } else {
        fprintf(stderr, "%s: Library functions did not return correct values\n", argv[0]);
        return 1;
    }
}
