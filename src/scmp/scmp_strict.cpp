//
// Created by Boxjan on Dec 25, 2018 15:26.
//

#include <string>
#include <seccomp.h>
#include <unistd.h>
#include <fcntl.h>

int load_strict_rule(const std::string &exec_path) {

        int syscall_whitelist[] = {
                // file system(IO)
                SCMP_SYS(read), SCMP_SYS(write),
                SCMP_SYS(close), SCMP_SYS(readlink),
                SCMP_SYS(writev), SCMP_SYS(readv),
                SCMP_SYS(flock), SCMP_SYS(fcntl),
                SCMP_SYS(fstat), SCMP_SYS(access),
                SCMP_SYS(lseek), SCMP_SYS(fsync),
                SCMP_SYS(lstat), SCMP_SYS(getdents),

                // system info
                SCMP_SYS(uname), SCMP_SYS(getrusage),
                SCMP_SYS(sysinfo), SCMP_SYS(getrlimit),
                SCMP_SYS(time), SCMP_SYS(getcwd),
                SCMP_SYS(clock_gettime),

                // memory
                SCMP_SYS(mmap), SCMP_SYS(munmap),
                SCMP_SYS(mremap), SCMP_SYS(brk),
                SCMP_SYS(mprotect), SCMP_SYS(madvise),

                // process
                SCMP_SYS(prctl), SCMP_SYS(arch_prctl),
                SCMP_SYS(exit_group), SCMP_SYS(exit),
                SCMP_SYS(rt_sigprocmask), SCMP_SYS(sigprocmask),
                SCMP_SYS(rt_sigaction), SCMP_SYS(sigaction),
                SCMP_SYS(prlimit64), SCMP_SYS(getpid),


                // system
                SCMP_SYS(poll), SCMP_SYS(stat),
                SCMP_SYS(getrandom),
        };

        int syscall_whitelist_len = sizeof(syscall_whitelist) / sizeof(int);

        scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_KILL & SCMP_ACT_LOG);

        if (!ctx) return -1;


        for (int i = 0; i < syscall_whitelist_len; i++) {
            if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, syscall_whitelist[i], 0) != 0) {
                return  -1;
            }
        }

        // add extra rule for execve
        if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(execve), 1, SCMP_A0(SCMP_CMP_EQ, (scmp_datum_t)(exec_path.c_str()))) != 0) {
            return -1;
        }
        // do not allow "w" and "rw"
        if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(open), 1, SCMP_CMP(1, SCMP_CMP_MASKED_EQ, O_WRONLY | O_RDWR, 0)) != 0) {
            return -1;
        }
        if (seccomp_rule_add(ctx, SCMP_ACT_ALLOW, SCMP_SYS(openat), 1, SCMP_CMP(2, SCMP_CMP_MASKED_EQ, O_WRONLY | O_RDWR, 0)) != 0) {
            return -1;
        }

        return seccomp_load(ctx) == 0 ? seccomp_release(ctx), 0 : -1;

}