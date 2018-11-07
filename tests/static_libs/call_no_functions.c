#define _X(a, b) a ## b
#define X(a, b) _X(a, b)
#define FN1 X(FUNCTION, 1)
#define FN2 X(FUNCTION, 2)

int FN1(int x)
{
    return x * 4;
}

int FN2(int x)
{
    return x * 5;
}
