//
// Created by Boxjan on Dec 25, 2018 15:25.
//

#include <seccomp.h>
#include <unistd.h>

int load_compile_rule() {
    int syscall_blacklist[] = {
            SCMP_SYS(socket),
            SCMP_SYS(setuid), SCMP_SYS(setgid),
            SCMP_SYS(setpgid), SCMP_SYS(setsid),
            SCMP_SYS(setreuid), SCMP_SYS(setregid),
            SCMP_SYS(setgroups), SCMP_SYS(setrlimit),
            SCMP_SYS(seccomp),
    };

    scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_ALLOW);

    if (!ctx) return -1;

    int syscall_blacklist_len = sizeof(syscall_blacklist) / sizeof(int);

    for (int i = 0; i < syscall_blacklist_len; i++) {
        if (seccomp_rule_add(ctx, SCMP_ACT_KILL & SCMP_ACT_LOG, syscall_blacklist[i], 0) != 0) {
            return  -1;
        }
    }

    return seccomp_load(ctx) == 0 ? seccomp_release(ctx), 0 : -1;
}