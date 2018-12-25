//
// Created by Boxjan on Dec 15, 2018 11:59.
//

#include <unistd.h>
#include <sys/types.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <pthread.h>
#include <sched.h>

#include "child.h"
#include "runner.h"
#include "log.h"

int run(const RuntimeConfig &config, RuntimeResult &result) {

    // check args
    if ( (config.max_cpu_time!= -1 && config.max_cpu_time < 1) || (config.max_memory != -1 && config.max_memory < 1) ||
            (config.max_stack != -1 && config.max_stack < 1) || (config.max_output_size != -1 && config.max_output_size < 1 ) ||
            (config.max_open_file_number != -1 && config.max_open_file_number < 1) || (config.uid != -1 && config.uid < 0) ||
            (config.gid != -1 && config.gid < 0) ) {
        log::error("procecc exit because %s", "bad args invalid");
        exit(1);
    }

    // you need root to set user/group id.
    if (config.uid != -1 || config.gid != -1) {
        if (getuid() != 0) {
            RUN_EXIT(NOT_RUNNING_BY_ROOT);
        }
    }

    struct timeval start_at, end_at;
    pid_t pid = fork();
    gettimeofday(&start_at, nullptr);

    if (pid == 0) {
        child(config);
        RUN_EXIT(CHILD_FAIL);
    } else if (pid < 0) {
        RUN_EXIT(FORK_FAIL);
    }

    // new thread to kill child if it spend to much time
    pthread_t timeout_tid = 0, memory_tid = 0;
    if (config.max_cpu_time != -1) {
        log::debug("timeout killer up");
        timeoutKillerStruct killerStruct(pid, config.max_cpu_time);
        if (pthread_create(&timeout_tid, nullptr, timeout_killer, (void *) (&killerStruct)) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(KILLER_THREAD_UP_FAIL);
        }

        if (pthread_detach(timeout_tid) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(THREAD_DETACH_FAIL);
        }
    }

    // new thread to kill child if it use to much memory
    if (config.max_memory != -1 && !config.use_rlimit_to_limit_memory) {
        log::debug("use killer to limit memory");
        log::debug("memory limit: %d bytes %d kb", config.max_memory * 1024, config.max_memory);
        memoryKillerStruct killerStruct(pid, config.max_memory);
        if (pthread_create(&memory_tid, nullptr, memory_killer, (void *) (&killerStruct)) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(KILLER_THREAD_UP_FAIL);
        }

        if (pthread_detach(memory_tid) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(THREAD_DETACH_FAIL);
        }
    }

    struct rusage usage;
    int status;
    if (wait4(pid, &status, WSTOPPED, &usage) == -1) {
        kill(pid, SIGKILL);
        RUN_EXIT(WAIT_ERROR);
    }
    gettimeofday(&end_at, nullptr);

    if (! (timeout_tid != 0 && pthread_kill(timeout_tid, 0) == ESRCH )) {
        pthread_cancel(timeout_tid);
    }

    result.status = status;
    result.cpu_time =
            (int) usage.ru_stime.tv_sec * 1000 + (int) usage.ru_stime.tv_usec / 1000 +
            (int) usage.ru_utime.tv_sec * 1000 + (int) usage.ru_utime.tv_usec / 1000;
    result.clock_time = (int) (end_at.tv_sec - start_at.tv_sec) * 1000 + (int) (end_at.tv_usec - start_at.tv_usec) / 1000;
    result.memory_use = (int) usage.ru_maxrss;

    getrusage(RUSAGE_CHILDREN, &usage);

    if (WIFSIGNALED(status)) {
        result.signal = WTERMSIG(status);
    }

    if (WIFEXITED(status)) {
        result.exit_code = WEXITSTATUS(status);
    }

    if (result.signal == SIGUSR2) {

        result.result = SYSTEM_ERROR;

    } else {

        if (result.exit_code != 0 || result.signal != 0 || result.status != 0) {
            result.result = RUNTIME_ERROR;
        }

        if (result.signal == SIGSYS) {
            result.result = RUNTIME_ERROR_BAD_SYSCALL;
        }

        if (config.max_cpu_time != -1 && ( result.status == 4991 || result.clock_time > config.max_cpu_time || result.cpu_time > config.max_cpu_time)) {
            result.result = TIME_LIMIT_EXCEEDED;
        }

        if (result.signal == SIGXFSZ) {
            result.result = OUTPUT_LIMIT_EXCEEDED;
        }

        if (result.signal == SIGSEGV && -1 != config.max_memory && result.memory_use > config.max_memory) {
            result.result = MEMORY_LIMIT_EXCEEDED;
        }

    }

    return 0;

}

void *timeout_killer(void *args) {
    auto *killer = (timeoutKillerStruct *)args;

    timespec delay = {killer->time / 1000, (killer->time % 1000 + 100) * 1000000};

    if (nanosleep(&delay, nullptr) != 0) {
        log::warn("It still have time, why the time out killer wake up?");
        kill(killer->pid, SIGKILL);
    }

    kill(killer->pid, SIGSTOP);

    return nullptr;

}

void *memory_killer(void *args) {
    auto *killer = (memoryKillerStruct *) args;
    int pagesize = getpagesize() / 1024;

    FILE *proc;
    char proc_file_path[1024];
    snprintf(proc_file_path, 1023, "/proc/%d/statm", killer->pid);

    char statm[512], *p;
    long mem[8];

    while (true) {
        timespec delay = {0, 1000};
        nanosleep(&delay, nullptr);

        if (kill(killer->pid, 0) == ESRCH) {
            break;
        }

        if (nullptr == (proc = fopen(proc_file_path, "r"))) {
            break;
        }
        fgets(statm, 511, proc);
        fclose(proc);

        p = statm;
        for (int i = 0; i < 7; i++) {
            mem[i] = strtol(p, &p, 10);
        }

        if (mem[1] * pagesize > killer->limit) {
            kill(killer->pid, SIGSEGV);
            break;
        }
    }

    return nullptr;
}