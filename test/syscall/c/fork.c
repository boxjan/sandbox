#include <unistd.h>
#include <stdio.h>

int main() {
    pid_t pid = fork();
    if (pid < 0) {
        printf("Fork Error!\n");
        exit(1);
    } 

    if (pid == 0) {
        printf("Father\n");
        return 0;
    } else {
        printf("Son\n");
        return 0;
    }
}