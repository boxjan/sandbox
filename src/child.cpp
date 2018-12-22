//
// Created by Boxjan on Dec 16, 2018 15:30.
//

#include <unistd.h>
#include <sys/resource.h>
#include <sys/time.h>
#include <signal.h>
#include <cstring>
#include <vector>
#include <iostream>

#include "child.h"
#include "runner.h"
#include "log.h"

const int MAX_ARGS = 128;
const int BUFFER_SIZE = 1024;

const int stdin_fd = fileno(stdin);
const int stdout_fd = fileno(stdout);
const int stderr_fd = fileno(stderr);

void close_file(FILE *fp) {
    if (fp == nullptr)
        return;

    int fd = fileno(fp);
    if (fd == stdin_fd || fd == stdout_fd || fd == stderr_fd)
        return;

    fclose(fp);
    fp = nullptr;
}

void child(const RuntimeConfig &config) {

    FILE *IN_FILE = nullptr, *OUT_FILE = nullptr, *ERR_FILE = nullptr;

    // set CPU time limit
    if (config.max_cpu_time != -1) {
        rlimit limit = {(rlim_t) (config.max_cpu_time + 1000) / 1000, (rlim_t) (config.max_cpu_time + 1000) / 1000};
        log::debug("cpu time limit: %d ms", config.max_cpu_time);
        if ( setrlimit(RLIMIT_CPU, &limit) != 0 ) {
            CHILD_EXIT(CPU_LIMIT_FAIL);
        }
    }

    // set memory limit
    if (config.max_memory != -1){
        rlimit limit = {(rlim_t) config.max_memory * 1024, (rlim_t) config.max_memory * 1024};
        log::debug("memory limit: %d bytes %d kb", config.max_memory * 1024, config.max_memory);
        if ( setrlimit(RLIMIT_AS, &limit) != 0 ) {
            CHILD_EXIT(MEMORY_LIMIT_FAIL);
        }
    }

    // set stack limit
    if (config.max_stack != -1){
        rlimit limit = {(rlim_t) config.max_stack * 1024, (rlim_t) config.max_stack * 1024};
        log::debug("stack limit: %d bytes %d kb", config.max_stack * 1024, config.max_stack);
        if (setrlimit(RLIMIT_STACK, &limit) != 0) {
            CHILD_EXIT(STACK_LIMIT_FAIL);
        }
    }

    // set output limit
    if (config.max_output_size != -1) {
        rlimit limit = {(rlim_t) config.max_output_size, (rlim_t) config.max_output_size};
        log::debug("output limit: %d bytes", config.max_output_size);
        if (setrlimit(RLIMIT_FSIZE, &limit) != 0) {
            CHILD_EXIT(OUTPUT_LIMIT_FAIL);
        }
    }

    // set open file number limit
    if (config.max_open_file_number != -1){
        rlimit limit = {(rlim_t) config.max_open_file_number, (rlim_t) config.max_open_file_number};
        log::debug("open file limit: %d", config.max_open_file_number);
        if (setrlimit(RLIMIT_NOFILE, &limit) != 0) {
            CHILD_EXIT(OPEN_FILE_COUNT_LIMIT_FAIL);
        }
    }

    // open input file and mount to stdin
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

    // open output file and mount to stdout
    if (config.output_path != "/dev/stdout") {
        log::debug("try to open output file: %s", config.output_path.c_str());
        OUT_FILE = fopen(config.output_path.c_str(), "w");
        if (OUT_FILE == nullptr) {
            CHILD_EXIT(OPEN_OUTPUT_FILE_FAIL);
        }

        if (dup2(fileno(OUT_FILE), stdout_fd) == -1) {
            CHILD_EXIT(MOUNT_OUTPUT_FILE_FAIL);
        }
    }

    // open error file and mount to stderr
    if (config.error_path != "/dev/stderr") {
        log::debug("try to open error file: %s", config.error_path.c_str());
        ERR_FILE = fopen(config.error_path.c_str(), "w");
        if (ERR_FILE == nullptr) {
            CHILD_EXIT(OPEN_ERROR_FILE_FAIL);
        }

        if (dup2(fileno(ERR_FILE), stderr_fd) == -1) {
            CHILD_EXIT(MOUNT_ERROR_FILE_FAIL);
        }
    }

    // set user id
    if (config.uid != -1){
        log::debug("set uid as: %d", config.uid);
        if (setuid((uid_t) config.uid) == -1) {
            CHILD_EXIT(SET_UID_FAIL);
        }
    }

    // set group id
    if (config.gid != -1)
    {
        log::debug("set gid as: %d", config.gid);
        if (setgid((gid_t) config.gid) == -1) {
            CHILD_EXIT(SET_GID_FAIL);
        }
    }

    // cut args
    char **args;
    {
        args = (char **)malloc(MAX_ARGS * sizeof(char *));
        memset((char*)args, 0, sizeof(args));

        int i = 0;
        char exec[BUFFER_SIZE];
        args[i++] = strncpy(exec, config.exec_path.c_str(), BUFFER_SIZE - 1);

        char *str = new char[config.exec_args.length() + 1];
        strcpy(str, config.exec_args.c_str());

        args[i] = strtok(str, " ");
        while (args[i++]) {
            args[i] = strtok(nullptr, " ");
        }
    }

    // cut env
    char **env = (char **)0;
    if (! config.exec_env.empty()) {
        env = (char **)malloc(MAX_ARGS * sizeof(char *));
        memset((char*)env, 0, sizeof(env));
        int i = 0;

        char *str = new char[config.exec_env.length() + 1];
        strcpy(str, config.exec_env.c_str());

        env[i] = strtok(str, " ");
        while (env[i++]) {
            env[i] = strtok(nullptr, " ");
        }
    }

    // load seccomp
    if (! config.scmp_name.empty()) {

    }

    if (! config.exec_env.empty()) {
        execve(config.exec_path.c_str(), args, env);
    } else {
        execv(config.exec_path.c_str(), args);
    }

    CHILD_EXIT(EXEC_ERROR);

}
