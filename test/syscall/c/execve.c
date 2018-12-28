#include <unistd.h>

int main() {
    execl("/bin/echo", "So bad", (void *)0);
    return 0;
}