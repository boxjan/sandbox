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
#include <cstdlib>
#include <cerrno>
#include <cstring>
#include <cstdio>

#include "child.h"
#include "runner.h"
#include "log.h"

int run(const RuntimeConfig *config, RuntimeResult *result) {

    Log::build(config->log_path, config->is_debug);

    // check args
    if ( (config->max_cpu_time!= -1 && config->max_cpu_time < 1) || (config->max_memory != -1 && config->max_memory < 1) ||
            (config->max_stack != -1 && config->max_stack < 1) || (config->max_output_size != -1 && config->max_output_size < 1 ) ||
            (config->max_open_file_number != -1 && config->max_open_file_number < 1) || (config->uid != -1 && config->uid < 0) ||
            (config->gid != -1 && config->gid < 0) ) {
        RUN_EXIT(ARGS_INVALID);
    }

    // you need root to set user/group id.
    if (config->uid != -1 || config->gid != -1) {
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

    LOG_DEBUG("Chile Pid: %d", pid);

    // new thread to kill child if it spend to much time
    pthread_t timeout_killer_tid = 0, memory_killer_tid = 0, thread_killer_tid = 0;
    if (config->max_time != -1) {
        killerStruct *timeout_killer_struct = new killerStruct{pid, config->max_time};
        if (pthread_create(&timeout_killer_tid, nullptr, timeout_killer, (void *) (timeout_killer_struct)) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(KILLER_THREAD_UP_FAIL);
        }

        if (pthread_detach(timeout_killer_tid) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(THREAD_DETACH_FAIL);
        }
    }

    // new thread to kill child if it use to much memory
    if (config->max_memory != -1 && !config->use_rlimit_to_limit_memory) {
        LOG_DEBUG("memory limit: %d bytes %d kb", config->max_memory * 1024, config->max_memory);
        killerStruct *memory_killer_struct = new killerStruct {pid, config->max_memory};
        if (pthread_create(&memory_killer_tid, nullptr, memory_killer, (void *) (memory_killer_struct)) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(KILLER_THREAD_UP_FAIL);
        }

        if (pthread_detach(memory_killer_tid) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(THREAD_DETACH_FAIL);
        }
    }

    // new thread to kill child if it use clone too much thread
   {
        int thread_limit = config->max_thread;
        LOG_DEBUG("use killer to limit thread num");
        if (thread_limit < 1) {
            LOG_INFO("Not set max thread, default 8");
            thread_limit = 8;
        }
        LOG_DEBUG("limit thread limit: %d", thread_limit);
        killerStruct *thread_killer_struct = new killerStruct{pid, thread_limit};
        if (pthread_create(&thread_killer_tid, nullptr, thread_killer, (void *) (thread_killer_struct)) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(KILLER_THREAD_UP_FAIL);
        }

        if (pthread_detach(thread_killer_tid) != 0) {
            kill(pid, SIGKILL);
            RUN_EXIT(THREAD_DETACH_FAIL);
        }
    }

    struct rusage usage{};
    int status;
    if (wait4(pid, &status, WSTOPPED, &usage) == -1) {
        kill(pid, SIGKILL);
        RUN_EXIT(WAIT_ERROR);
    }
    gettimeofday(&end_at, nullptr);

    result->status = status;
    result->cpu_time =
            (int) usage.ru_stime.tv_sec * 1000 + (int) usage.ru_stime.tv_usec / 1000 +
            (int) usage.ru_utime.tv_sec * 1000 + (int) usage.ru_utime.tv_usec / 1000;
    result->clock_time = (int) (end_at.tv_sec - start_at.tv_sec) * 1000 + (int) (end_at.tv_usec - start_at.tv_usec) / 1000;
    result->memory_use = (int) usage.ru_maxrss;

    getrusage(RUSAGE_CHILDREN, &usage);

    if (WIFSIGNALED(status)) {
        result->signal = WTERMSIG(status);
    }

    if (WIFEXITED(status)) {
        result->exit_code = WEXITSTATUS(status);
    }

    if (result->signal == SIGUSR2) {

        result->result = SYSTEM_ERROR;

    } else {

        if (result->exit_code != 0 || result->signal != 0 || result->status != 0) {
            result->result = RUNTIME_ERROR;
        }

        if (result->signal == SIGSYS) {
            result->result = RUNTIME_ERROR_BAD_SYSCALL;
        }

        if (config->max_cpu_time != -1 && ( result->status == 4991 || result->clock_time > config->max_cpu_time || result->cpu_time > config->max_cpu_time)) {
            result->result = TIME_LIMIT_EXCEEDED;
        }

        if (result->signal == SIGXFSZ) {
            result->result = OUTPUT_LIMIT_EXCEEDED;
        }

        if (result->signal == SIGSEGV && -1 != config->max_memory && result->memory_use > config->max_memory) {
            result->result = MEMORY_LIMIT_EXCEEDED;
        }
    }

    return 0;

}

void *timeout_killer(void *args) {
    auto *killer = static_cast<killerStruct*>(args);
    if (killer->pid == 0) {
        LOG_ERROR("Timeout killer can not get pid");
    } else {
        LOG_DEBUG("timeout killer up");

        timespec delay = {killer->limit / 1000, (killer->limit % 1000 + 100) * 1000000};

        if (nanosleep(&delay, nullptr) != 0) {
            LOG_WARN("It still have time, why the time out killer wake up?");
            kill(killer->pid, SIGKILL);
        }

        if (kill(killer->pid, 0) != ESRCH) {
            LOG_WARN("Timeout Kill Work!");
            kill(killer->pid, SIGSTOP);
        }

    }

    delete killer;
    return nullptr;


}

void *memory_killer(void *args) {
    auto *killer = static_cast<killerStruct*>(args);

    if (killer->pid == 0) {
        LOG_ERROR("Memory killer can not get pid");
    } else {
        int pagesize = getpagesize() / 1024;
        LOG_DEBUG("memory killer up");

        FILE *proc;
        char proc_file_path[1024];
        snprintf(proc_file_path, 1023, "/proc/%d/statm", (int) killer->pid);

        char statm[512], *p;
        long mem[8];

        timespec delay{};

        while (true) {

            if (nullptr == (proc = fopen(proc_file_path, "r"))) {
                if (kill(killer->pid, 0) == 0) {
                    LOG_WARN("Can not open %s", proc_file_path);
                }
                break;
            }

            p = fgets(statm, 511, proc);
            fclose(proc);

            for (int i = 0; i < 7; i++) mem[i] = strtol(p, &p, 10);

            if (mem[1] * pagesize > killer->limit) {
                LOG_WARN("Memory Kill Work!");
                kill(killer->pid, SIGSEGV);
                break;
            }

            delay = {0, 1000};
            nanosleep(&delay, nullptr);
        }
    }

    delete killer;
    return nullptr;
}

void *thread_killer(void *args) {
    auto *killer = static_cast<killerStruct*>(args);
    if (killer->pid == 0) {
        LOG_ERROR("Thread killer can not get pid");
    } else {

        LOG_DEBUG("thread killer up");

        FILE *proc;
        char proc_file_path[1024];
        snprintf(proc_file_path, 1023, "/proc/%d/status", (int) killer->pid);

        timespec delay{};
        char line[1024];
        int thread_count;

        while (true) {

            if (nullptr == (proc = fopen(proc_file_path, "r"))) {
                if (kill(killer->pid, 0) == 0) {
                    LOG_WARN("Can not open %s", proc_file_path);
                }
                break;
            }

            while (feof(proc) == 0) {
                if (nullptr == fgets(line, 1023, proc) && kill(killer->pid, 0) == 0) {
                    if (errno != 0) {
                        int eno = errno;
                        LOG_ERROR("Try to read proc file error! Errno: %d", eno);
                        kill(killer->pid, SIGKILL);

                        delete killer;
                        return nullptr;
                    }
                }

                if (line[0] != 'T') continue;

                if (strstr(line, "Threads") != nullptr) break;
            }

            fclose(proc);

            if (sscanf(line, "%*s %d", &thread_count) == 0) {
                LOG_WARN("Try to sscanf from `%d` error!", line);
                continue;
            }

            if (thread_count > killer->limit) {
                LOG_WARN("Thread Kill Work!");
                kill(killer->pid, SIGKILL);
                break;
            }

            delay = {0, 1000};
            nanosleep(&delay, nullptr);
        }
    }

    delete killer;
    return nullptr;

}