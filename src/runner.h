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
    int max_open_file_number;

    string exec_path;
    string exec_env;
    string exec_args;

    string input_path;
    string output_path;
    string error_path;

    string log_path;
    string scmp_name;

    int uid;
    int gid;
};

enum RUN_EXIT_CODE {
    NOT_RUNNING_BY_ROOT,
    CHILD_FAIL,
};

const char RUN_EXIT_REASON[][32] = {
        "NOT RUNNING BY ROOT",
        "CHILE PROCESS FAIL",
};     //12345678123456781234567812345678

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

int run(const RuntimeConfig &config, RuntimeResult &result);

#define RUN_EXIT(code) log::error("procecc exit because %s", RUN_EXIT_REASON[code]);


#endif //SANDBOX_RUNNER_H
