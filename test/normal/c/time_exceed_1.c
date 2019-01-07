#include <stdio.h>

int main() {
    long long n = 0x0fffffffffffffff;

    while (n) {
        n --;
    }

    printf("So bad");
    return 0;
}