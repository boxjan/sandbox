//
// Created by Boxjan on Dec 15, 2018 11:59.
//

#include <unistd.h>
#include <sys/types.h>
#include <sys/time.h>
#include <sys/resource.h>
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
        return 0;
    } else if (pid < 0) {
        RUN_EXIT(FORK_FAIL);
    }

    // new thread to kill child if it spend to much time
    pthread_t tid = 0;
    if (config.max_cpu_time != -1) {
        timeoutKillerStruct killerStruct(pid, config.max_cpu_time);
        if (pthread_create(&tid, nullptr, timeout_killer, (void *) (&killerStruct)) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(KILLER_THREAD_UP_FAIL);
        }

        if (pthread_detach(tid) != 0) {
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

    if (tid != 0 && pthread_kill(tid, 0) == ESRCH ) {
        result.result = TIME_LIMIT_EXCEEDED;
    } else {
        pthread_cancel(tid);
    }

    result.cpu_time =
            (int) usage.ru_stime.tv_sec * 1000 + (int) usage.ru_stime.tv_usec / 1000 +
            (int) usage.ru_utime.tv_sec * 1000 + (int) usage.ru_utime.tv_usec / 1000;
    result.clock_time = (int) (end_at.tv_sec - start_at.tv_sec) * 1000 + (int) (end_at.tv_usec - start_at.tv_usec) / 1000;

    result.memory_use = (int) usage.ru_maxrss;

    if (WIFSIGNALED(status)) {
        result.signal = WTERMSIG(status);
    }

    if (result.signal == SIGUSR2) {
        result.result = SYSTEM_ERROR;
        return 0;
    }

    if (result.signal == SIGSEGV ) {
        if (config.max_memory != -1 && result.memory_use > config.max_memory) {
            result.result = MEMORY_LIMIT_EXCEEDED;
        } else {
            result.result = RUNTIME_ERROR;
        }

    }

    if (config.max_cpu_time != -1 && result.cpu_time > config.max_cpu_time) {
        result.result = TIME_LIMIT_EXCEEDED;
    }

    return 0;

}

void *timeout_killer(void *args) {
    timeoutKillerStruct *killer = (timeoutKillerStruct *)args;

    struct timespec delay = {killer->time / 1000, (killer->time % 1000 + 300) * 1000}, remainder;

    if (nanosleep(&delay, &remainder) != 0) {
        kill(killer->pid, SIGKILL);
//        RUN_EXIT(KILLER_WAKEUP);
    }

    kill(killer->pid, SIGUSR1);

    return nullptr;

}