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

struct RuntimeResult {
    int cpu_time;
    int cpu_user_time;
    int cpu_core_time;
    int memory_use;
    int exit_code;
};

int run(const RuntimeConfig &config, RuntimeResult &result);
#endif //SANDBOX_RUNNER_H
