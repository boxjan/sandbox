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
    arg.add<int>("max_cpu_time", 't', "set cpu time limit in micro seconds(ms)", false, -1);
    arg.add<int>("max_stack", 's', "set process stack limit(kb)", false, -1);
    arg.add<int>("max_memory", 'm', "set memory limit(kb)", false, -1);
    arg.add<int>("max_output_size", 'q', "set output limit(byte)", false, -1);
    arg.add<int>("max_open_file_number", 'f', "set program open file number limit", false, -1);

    arg.add<string>("exec_path", 'c', "set executable file path", true);
    arg.add<string>("exec_args", 'a', "set exec arg, if have more than one args, use quotes", false);
    arg.add<string>("exec_env", 'n', "set exec environment, if have more than one args ,use quotes", false);

    arg.add<string>("input_path", 'i', "set input redirect", false, "/dev/stdin");
    arg.add<string>("output_path", 'o', "set output redirect", false, "/dev/stdout");
    arg.add<string>("error_path", 'e', "set error output redirect", false, "/dev/stderr");

    arg.add<int>("uid", 'u', "set running user id", false, -1);
    arg.add<int>("gid", 'g', "set running group id", false, -1);
    arg.add<string>("scmp_name", 'p', "set running seccomp rule name", false, "", cmdline::oneof<string>("", "low", "mid", "high"));

    arg.add<string>("log_path", 'l', "set runtime log path", false, "/dev/stderr");
    arg.add("verbose", 'v', "record log in verbose");

    arg.footer("\nSandbox design for OnlineJudge \nNotice: If there are multiple parameters , please include it in quotation marks just like --exec_args \"-l 1\"\n");

    arg.parse_check(argc, argv);

    RuntimeResult result;
    RuntimeConfig config;

    if ( arg.exist("log_path") ) {
        std::string path;
        config.log_path = arg.get<string>("log_path");
        Log::openFile(path.c_str());
    }

    if ( arg.exist("verbose") ) {
        log::isDebug();
    }

    // set config
    config.max_cpu_time = arg.get<int>("max_cpu_time");

    config.max_stack = arg.get<int>("max_stack");

    config.max_memory = arg.get<int>("max_memory");

    config.max_output_size = arg.get<int>("max_output_size");

    config.max_open_file_number = arg.get<int>("max_open_file_number");

    config.exec_path = arg.get<string>("exec_path");

    config.exec_args = arg.get<string>("exec_args");

    config.exec_env = arg.get<string>("exec_env");

    config.input_path = arg.get<string>("input_path");

    config.output_path = arg.get<string>("output_path");

    config.error_path = arg.get<string>("error_path");

    config.uid = arg.get<int>("uid");

    config.gid = arg.get<int>("gid");

    config.scmp_name = arg.get<string>("scmp_name");

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
