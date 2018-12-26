//
// Created by Boxjan on Dec 25, 2018 15:38.
//

#include <seccomp.h>
#include <unistd.h>
#include <fcntl.h>
#include <string>

int load_gentle_rule(const std::string &exec_path) {
    int syscall_blacklist[] = {
            SCMP_SYS(socket),
            SCMP_SYS(setuid), SCMP_SYS(setgid),
            SCMP_SYS(setpgid), SCMP_SYS(setsid),
            SCMP_SYS(setreuid), SCMP_SYS(setregid),
            SCMP_SYS(setgroups), SCMP_SYS(setrlimit),
            SCMP_SYS(vfork), SCMP_SYS(fork),
            SCMP_SYS(chmod), SCMP_SYS(chown),
            SCMP_SYS(chown32), SCMP_SYS(fchmod),
            SCMP_SYS(fchown), SCMP_SYS(fchownat),
            SCMP_SYS(link), SCMP_SYS(shutdown),
            SCMP_SYS(seccomp), SCMP_SYS(rmdir),
            SCMP_SYS(rename),
            };

    scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_ALLOW);

    if (!ctx) return -1;

    int syscall_blacklist_len = sizeof(syscall_blacklist) / sizeof(int);

    for (int i = 0; i < syscall_blacklist_len; i++) {
        if (seccomp_rule_add(ctx, SCMP_ACT_KILL & SCMP_ACT_LOG, syscall_blacklist[i], 0) != 0) {
            return  -1;
        }
    }

    if (seccomp_rule_add(ctx, SCMP_ACT_KILL, SCMP_SYS(execve), 1, SCMP_A0(SCMP_CMP_NE, (scmp_datum_t)(exec_path.c_str()))) != 0) {
        return -1;
    }

    if (seccomp_rule_add(ctx, SCMP_ACT_KILL, SCMP_SYS(open), 1, SCMP_CMP(1, SCMP_CMP_MASKED_EQ, O_WRONLY, O_WRONLY)) != 0) {
        return -1;
    }
    if (seccomp_rule_add(ctx, SCMP_ACT_KILL, SCMP_SYS(open), 1, SCMP_CMP(1, SCMP_CMP_MASKED_EQ, O_RDWR, O_RDWR)) != 0) {
        return -1;
    }
    // do not allow "w" and "rw" using openat
    if (seccomp_rule_add(ctx, SCMP_ACT_KILL, SCMP_SYS(openat), 1, SCMP_CMP(2, SCMP_CMP_MASKED_EQ, O_WRONLY, O_WRONLY)) != 0) {
        return -1;
    }
    if (seccomp_rule_add(ctx, SCMP_ACT_KILL, SCMP_SYS(openat), 1, SCMP_CMP(2, SCMP_CMP_MASKED_EQ, O_RDWR, O_RDWR)) != 0) {
        return -1;
    }


    return seccomp_load(ctx) == 0 ? seccomp_release(ctx), 0 : -1;
}