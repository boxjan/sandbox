//
// Created by Boxjan on Dec 16, 2018 15:30.
//

#include <unistd.h>
#include <sys/resource.h>
#include <sys/time.h>

#include "child.h"
#include "log.h"

const int stdin_fd = fileno(stdin);
const int stdout_fd = fileno(stdout);
const int stderr_fd = fileno(stderr);

void close_file(FILE *fp) {
    int fd = fileno(fp);
    if (fp != nullptr || fd != stdin_fd || fd != stdout_fd || fd != stderr_fd)
        return;

    fclose(fp);
    fp = nullptr;
}

void child(const RuntimeConfig &config) {

    FILE *IN_FILE = nullptr, *OUT_FILE = nullptr, *ERR_FILE = nullptr;

    // set CPU time limit
    {
        rlimit limit;
        log::debug("cpu time limit: %d ms", config.max_cpu_time);
        limit.rlim_max = limit.rlim_cur = (rlim_t) (config.max_cpu_time + 500) / 1000;
        if ( setrlimit(RLIMIT_CPU, &limit) != 0 ) {
            CHILD_EXIT(CPU_LIMIT_FAIL);
        }
    }

    // set memory limit
    {
        rlimit limit;
        log::debug("memory limit: %d bytes", config.max_memory);
        limit.rlim_max = limit.rlim_cur = (rlim_t) config.max_memory;
        if ( setrlimit(RLIMIT_AS, &limit) != 0 ) {
            CHILD_EXIT(MEMORY_LIMIT_FAIL);
        }
    }

    // set stack limit
    {
        rlimit limit;
        log::debug("stack limit: %d bytes", config.max_stack);
        limit.rlim_max = limit.rlim_cur = (rlim_t) config.max_stack;
        if (setrlimit(RLIMIT_STACK, &limit) != 0) {
            CHILD_EXIT(STACK_LIMIT_FAIL);
        }
    }

    // set output limit
    {
        rlimit limit;
        log::debug("output limit: %d bytes", config.max_output_size);
        limit.rlim_max = limit.rlim_cur = (rlim_t) config.max_output_size;
        if (setrlimit(RLIMIT_FSIZE, &limit) != 0) {
            CHILD_EXIT(OUTPUT_LIMIT_FAIL);
        }
    }

    // set open file number limit
    {
        rlimit limit;
        log::debug("open file limit: %d ms", config.max_open_file_number);
        limit.rlim_max = limit.rlim_cur = (rlim_t) config.max_open_file_number;
        if (setrlimit(RLIMIT_NOFILE, &limit) != 0) {
            CHILD_EXIT(OPEN_FILE_COUNT_LIMIT_FAIL);
        }
    }

    // open input file and mount to stdin
    {
        if (config.input_path != "/dev/stdin") {
            log::debug("try to open input file: %s", config.input_path.c_str());
            IN_FILE = fopen(config.input_path.c_str(), "r");
            if (IN_FILE == nullptr) {
                CHILD_EXIT(OPEN_INPUT_FILE_FAIL);
            }

            if (dup2(fileno(IN_FILE), stdin_fd) == -1) {
                CHILD_EXIT(MOUNT_INPUT_FILE_FAIL);
            }
        }
    }

    // open output file and mount to stdout
    {
        if (config.output_path != "/dev/stdout") {
            log::debug("try to open input file: %s", config.output_path.c_str());
            IN_FILE = fopen(config.output_path.c_str(), "w");
            if (IN_FILE == nullptr) {
                CHILD_EXIT(OPEN_OUTPUT_FILE_FAIL);
            }

            if (dup2(fileno(IN_FILE), stdout_fd) == -1) {
                CHILD_EXIT(MOUNT_OUTPUT_FILE_FAIL);
            }
        }
    }

    // open error file and mount to stderr
    {
        if (config.output_path != "/dev/stdout") {
            log::debug("try to open input file: %s", config.output_path.c_str());
            IN_FILE = fopen(config.output_path.c_str(), "w");
            if (IN_FILE == nullptr) {
                CHILD_EXIT(OPEN_ERROR_FILE_FAIL);
            }

            if (dup2(fileno(IN_FILE), stderr_fd) == -1) {
                CHILD_EXIT(MOUNT_ERROR_FILE_FAIL);
            }
        }

    }

    // set user id
    {
        log::debug("set uid as: %d", config.uid);
        if (setuid((uid_t) config.uid) == -1) {
            CHILD_EXIT(SET_UID_FAIL);
        }
    }

    // set group id
    {
        log::debug("set gid as: %d", config.gid);
        if (setgid((gid_t) config.gid) == -1) {
            CHILD_EXIT(SET_GID_FAIL);
        }
    }

    // cut args

    // cut env

    // load seccomp


//    execve(config.exec_path.c_str(), );
}
