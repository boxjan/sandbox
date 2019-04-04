//
// Created by Boxjan on Dec 15, 2018 11:59.
//

#ifndef SANDBOX_RUNNER_H
#define SANDBOX_RUNNER_H

struct RuntimeConfig {
    int max_time;
    int max_cpu_time;
    int max_stack;
    int max_memory;
    int max_output_size;
    int max_open_file_number;
    int max_thread;

    char *exec_path;
    char *exec_env;
    char *exec_args;

    char *input_path;
    char *output_path;
    char *error_path;
    char use_rlimit_to_limit_memory;

    char * log_path;
    char is_debug;
    char *scmp_name;

    int uid;
    int gid;
};

enum RUN_EXIT_CODE {
    ARGS_INVALID,
    NOT_RUNNING_BY_ROOT,
    FORK_FAIL,
    WAIT_ERROR,
    KILLER_THREAD_UP_FAIL,
    THREAD_DETACH_FAIL,
    CHILD_FAIL,
};

const char RUN_EXIT_REASON[][32] = {
        "ARGS INVALID",
        "NOT RUNNING BY ROOT",
        "FORK FAIL",
        "WAIT PID ERROR",
        "KILLER THREAD UP FAIL",
        "THREAD DETACH FAIL",
        "CHILE PROCESS FAIL",
};     //12345678123456781234567812345678

enum RESULT {
    SUCCESS_EXIT,
    TIME_LIMIT_EXCEEDED,
    MEMORY_LIMIT_EXCEEDED,
    OUTPUT_LIMIT_EXCEEDED,
    RUNTIME_ERROR,
    RUNTIME_ERROR_BAD_SYSCALL,
    SYSTEM_ERROR,
};

const char RESULT_STRING[][32] = {
        "SUCCESS EXIT",
        "TIME LIMIT EXCEEDED",
        "MEMORY LIMIT EXCEEDED",
        "OUTPUT LIMIT EXCEEDED",
        "RUNTIME ERROR",
        "BAD SYSTEM CALL",
        "SYSTEM ERROR",
};

struct RuntimeResult {
    int cpu_time;
    int clock_time;
    int memory_use;
    int exit_code;
    int signal;
    int result;
    int status;

    RuntimeResult() {
        cpu_time = 0;
        clock_time = 0;
        memory_use = 0;
        exit_code = 0;
        signal = 0;
        result = SUCCESS_EXIT;
        status = 0;
    };
};

struct killerStruct {
    pid_t pid;
    int limit;
};


void *timeout_killer(void*);
void *memory_killer(void*);
void *thread_killer(void*);

#ifdef __cplusplus
    extern "C" {
#endif
    int run(const RuntimeConfig *config, RuntimeResult *result);
#ifdef __cplusplus
}
#endif



#define RUN_EXIT(code) { \
result->result = SYSTEM_ERROR ; \
LOG_ERROR("procecc exit because %s", RUN_EXIT_REASON[code]); return -1; \
}


#endif //SANDBOX_RUNNER_H
