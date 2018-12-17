//
// Created by Boxjan on Dec 15, 2018 11:59.
//

#ifndef SANDBOX_RUNNER_H
#define SANDBOX_RUNNER_H
#include <string>
using std::string;

struct RuntimeConfig {
    int max_cpu_time;
    int max_stack;
    int max_memory;
    int max_output_size;
    int max_process_number;
    int max_open_file_number;

    string exec_path;
    string env;
    string exec_args;

    string input_path;
    string output_path;
    string error_path;

    string log_path;

    int uid;
    int gid;
} ;

enum CHILD_EXIT_REASON {

};

enum RESULT {
    SUCCESS_EXIT,
    TIME_LIMIT_EXCEEDED,
    CORE_TIME_LIMIT_EXCEEDED,
    USER_TIME_LIMIT_EXCEEDED,
    MEMORY_LIMIT_EXCEEDED,
    RUNTIME_ERROR_NOT_ALLOW_CALL,
    RUNTIME_ERROR_OUT_OF_BOUNDS,
    SYSTEM_ERROR,
};

struct RuntimeResult {
    int cpu_time;
    int cpu_user_time;
    int cpu_core_time;
    int memory_use;
    int exit_code;
    int signal;
    int result;

    RuntimeResult() {
        cpu_time = 0;
        cpu_user_time = 0;
        cpu_core_time = 0;
        memory_use = 0;
        exit_code = 0;
        signal = 0;
        result = SUCCESS_EXIT;

    };
};

struct runtime {
    FILE *in;
    int in_fd;
    FILE *out;
    int out_fd;
    FILE *err;
    int err_fd;
};


int run(const RuntimeConfig &config, RuntimeResult &result);
#endif //SANDBOX_RUNNER_H
