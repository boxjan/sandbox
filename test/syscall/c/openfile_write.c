#include <stdio.h>

int main() {
    FILE *fp = fopen("write_file", "w");
    if (fp != NULL) {
        fprintf("That is so bad~");
    }

    fclose(fp);
}