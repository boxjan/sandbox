//
// Created by hzj on 10:00 Dec 15, 2018
//
#include <iostream>
#include <string>

#include "cmdline.h"
#include "runner.h"
#include "log.h"

int main(int argc, char **argv) {
    using std::cout;

    cmdline::parser arg;
    arg.add<int>("max_cpu_time", 't', "set cpu time limit in micro seconds(ms)", false);
    arg.add<int>("max_stack", 's', "set process stack limit(kb)", false);
    arg.add<int>("max_memory", 'm', "set memory limit(kb)", false);
    arg.add<int>("max_output_size", 'q', "set output limit(byte)", false);
    arg.add<int>("max_open_file_number", 'f', "set program open file number limit", false);

    arg.add<string>("exec_path", 'c', "set executable file path", true);
    arg.add<string>("exec_args", 'a', "set exec arg, if have more than one args, use quotes", false);
    arg.add<string>("exec_env", 'n', "set exec environment, if have more than one args ,use quotes", false);

    arg.add<string>("input_path", 'i', "set input redirect", false);
    arg.add<string>("output_path", 'o', "set output redirect", false);
    arg.add<string>("error_path", 'e', "set error output redirect", false);

    arg.add<int>("uid", 'u', "set running user id", false);
    arg.add<int>("gid", 'g', "set running group id", false);
    arg.add<string>("scmp_rule_name", 'p', "set running seccomp rule name", false);
    arg.add("no_change_child_id", 0, "if you run by root, with this will let child run by root too.");
    arg.add("use_rlimit_to_limit_memory", 0, "use rlimit to limit memory");

    arg.add<string>("log_path", 'l', "set runtime log path", false);
    arg.add("verbose", 'v', "record log in verbose");

    arg.footer("\nSandbox design for OnlineJudge \nNotice: If there are multiple parameters , please include it in quotation marks just like --exec_args \"-l 1\"\n");

    arg.parse_check(argc, argv);

    RuntimeResult result;
    RuntimeConfig config;

    config.log_path = arg.exist("log_path") ? arg.get<string>("log_path") : "/dev/stderr";
    config.is_debug = arg.exist("verbose");

    // set config
    config.max_cpu_time = arg.exist("max_cpu_time") ? arg.get<int>("max_cpu_time") : -1;

    config.max_stack = arg.exist("max_stack") ? arg.get<int>("max_stack") : -1;

    config.max_memory = arg.exist("max_memory") ? arg.get<int>("max_memory") : -1;

    config.max_output_size = arg.exist("max_output_size") ? arg.get<int>("max_output_size") : -1;

    config.max_open_file_number = arg.exist("max_open_file_number") ? arg.get<int>("max_open_file_number") : -1;

    config.exec_path = arg.get<string>("exec_path");

    config.exec_args = arg.exist("exec_args") ? arg.get<string>("exec_args") : "";

    config.exec_env = arg.exist("exec_env") ? arg.get<string>("exec_env") : "";

    config.input_path = arg.exist("input_path") ? arg.get<string>("input_path") : "/dev/stdin";

    config.output_path = arg.exist("output_path") ? arg.get<string>("output_path") : "/dev/stdout";

    config.error_path = arg.exist("error_path") ? arg.get<string>("error_path") : "/dev/stderr";

    config.uid = arg.exist("uid") ? arg.get<int>("uid") : -1;

    config.gid = arg.exist("gid") ? arg.get<int>("gid") : -1;

    config.scmp_name = arg.exist("scmp_rule_name") ? arg.get<string>("scmp_rule_name") : "";

    config.use_rlimit_to_limit_memory =  arg.exist("use_rlimit_to_limit_memory");


    // if you running by root, for safe will change child uid and gid;
    if (getuid() == 0 && ! arg.exist("no_change_child_id")) {
        if (config.uid == -1) {
            config.uid = 65534;
        }
        if (config.gid == -1) {
            config.gid = 65534;
        }
    }

    run(config, result);

    printf("{\n"
           "  \"CPU_TIME\": %d,\n"
           "  \"CLOCK_TIME\": %d,\n"
           "  \"MEMORY\": %d,\n"
           "  \"STATUS\": %d,\n"
           "  \"SIGNAL:\" %d,\n"
           "  \"EXIT_CODE:\": %d,\n"
           "  \"RESULT_CODE\": %d,\n"
           "  \"RESULT\": \"%s\"\n"
           "}\n" ,
           result.cpu_time, result.clock_time,
           result.memory_use, result.status,
           result.signal, result.exit_code,
           result.result, RESULT_STRING[result.result]);
}
