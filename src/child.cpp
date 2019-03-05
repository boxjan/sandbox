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
#include "scmp/scmp.h"

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
        LOG_DEBUG("cpu time limit: %d ms", config.max_cpu_time);
        if ( setrlimit(RLIMIT_CPU, &limit) != 0 ) {
            CHILD_EXIT(CPU_LIMIT_FAIL);
        }
    }

    // set memory limit
    if (config.use_rlimit_to_limit_memory  && config.max_memory != -1){
        rlimit limit = {(rlim_t) config.max_memory * 1024, (rlim_t) config.max_memory * 1024};
        LOG_DEBUG("use setrlimit to limit memory");
        LOG_DEBUG("memory limit: %d bytes %d kb", config.max_memory * 1024, config.max_memory);
        if ( setrlimit(RLIMIT_AS, &limit) != 0 ) {
            CHILD_EXIT(MEMORY_LIMIT_FAIL);
        }
    }

    // set stack limit
    if (config.max_stack != -1){
        rlimit limit = {(rlim_t) config.max_stack * 1024, (rlim_t) config.max_stack * 1024};
        LOG_DEBUG("stack limit: %d bytes %d kb", config.max_stack * 1024, config.max_stack);
        if (setrlimit(RLIMIT_STACK, &limit) != 0) {
            CHILD_EXIT(STACK_LIMIT_FAIL);
        }
    }

    // set output limit
    if (config.max_output_size != -1) {
        rlimit limit = {(rlim_t) config.max_output_size, (rlim_t) config.max_output_size};
        LOG_DEBUG("output limit: %d bytes", config.max_output_size);
        if (setrlimit(RLIMIT_FSIZE, &limit) != 0) {
            CHILD_EXIT(OUTPUT_LIMIT_FAIL);
        }
    }

    // set open file number limit
    if (config.max_open_file_number != -1){
        rlimit limit = {(rlim_t) config.max_open_file_number, (rlim_t) config.max_open_file_number};
        LOG_DEBUG("open file limit: %d", config.max_open_file_number);
        if (setrlimit(RLIMIT_NOFILE, &limit) != 0) {
            CHILD_EXIT(OPEN_FILE_COUNT_LIMIT_FAIL);
        }
    }

    // open input file and mount to stdin
    if (strcmp(config.input_path, "/dev/stdin") != 0) {
        LOG_DEBUG("try to open input file: %s", config.input_path);
        IN_FILE = fopen(config.input_path, "r");
        if (IN_FILE == nullptr) {
            CHILD_EXIT(OPEN_INPUT_FILE_FAIL);
        }

        if (dup2(fileno(IN_FILE), stdin_fd) == -1) {
            CHILD_EXIT(MOUNT_INPUT_FILE_FAIL);
        }
    }

    // open output file and mount to stdout
    if (strcmp(config.output_path, "/dev/stdout") != 0) {
        LOG_DEBUG("try to open output file: %s", config.output_path);
        OUT_FILE = fopen(config.output_path, "w");
        if (OUT_FILE == nullptr) {
            CHILD_EXIT(OPEN_OUTPUT_FILE_FAIL);
        }

        if (dup2(fileno(OUT_FILE), stdout_fd) == -1) {
            CHILD_EXIT(MOUNT_OUTPUT_FILE_FAIL);
        }
    }

    // open error file and mount to stderr
    if (strcmp(config.error_path, "/dev/stderr") != 0) {
        LOG_DEBUG("try to open error file: %s", config.error_path);
        ERR_FILE = fopen(config.error_path, "w");
        if (ERR_FILE == nullptr) {
            CHILD_EXIT(OPEN_ERROR_FILE_FAIL);
        }

        if (dup2(fileno(ERR_FILE), stderr_fd) == -1) {
            CHILD_EXIT(MOUNT_ERROR_FILE_FAIL);
        }
    }

    // set group id
    if (config.gid != -1)
    {
        LOG_DEBUG("set gid as: %d", config.gid);
        if (setgid((gid_t) config.gid) == -1) {
            CHILD_EXIT(SET_GID_FAIL);
        }
    }

    // set user id
    if (config.uid != -1){
        LOG_DEBUG("set uid as: %d", config.uid);
        if (setuid((uid_t) config.uid) == -1) {
            CHILD_EXIT(SET_UID_FAIL);
        }
    }

    if (config.uid != -1 || config.gid != -1) {
        rlimit limit = {512, 768};
        if (setrlimit(RLIMIT_NPROC, &limit) !=0) {
            CHILD_EXIT(OTHER_FAIL);
        }
    }

    // cut args
    char **args;
    {
        args = (char **)malloc(MAX_ARGS * sizeof(char *));
        memset((char*)args, 0, sizeof(args));

        int i = 0;
        char exec[BUFFER_SIZE];
        args[i++] = strncpy(exec, config.exec_path, BUFFER_SIZE - 1);

        char *str = new char[strlen(config.exec_args) + 1];
        strcpy(str, config.exec_args);

        args[i] = strtok(str, " ");
        while (args[i++]) {
            args[i] = strtok(nullptr, " ");
        }
    }

    // cut env
    char **env = (char **) nullptr;
    if ( strlen(config.exec_env) != 0) {
        env = (char **)malloc(MAX_ARGS * sizeof(char *));
        memset((char*)env, 0, sizeof(env));
        int i = 0;

        char *str = new char[strlen(config.exec_env) + 1];
        strcpy(str, config.exec_env);

        env[i] = strtok(str, " ");
        while (env[i++]) {
            env[i] = strtok(nullptr, " ");
        }
    }

    // load seccomp
    if (strlen(config.scmp_name) != 0) {
        LOG_DEBUG("load %s level seccomp rule", config.scmp_name);
        if (load(config.scmp_name, config.exec_path) == -1) {
            CHILD_EXIT(SCMP_LOAD_FAIL);
        }

    }

    if (strlen(config.exec_env) != 0) {
        execvpe(config.exec_path, args, env);
    } else {
        execvp(config.exec_path, args);
    }

    CHILD_EXIT(EXEC_ERROR);

}
