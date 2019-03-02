#include <stdio.h>

int main() {
    int a = 102400000;
    while (a--) {
        int *p = malloc(sizeof(int));
    }
    printf("So bad!");
    return 0;
}