//
// Created by hzj on 10:00 Dec 15, 2018
//
#include <iostream>
#include <string>

#include "argh.h"
#include "runner.h"
#include "log.h"
#include <seccomp.h>


const char *version = "0.0.1";

void showVersion();

void showHelp();

int main(int argc, char **argv) {
    using std::cout;
    argh::parser arg;

    arg.parse(argc, argv, argh::parser::PREFER_PARAM_FOR_UNREG_OPTION);

    if ( arg({"-h", "--help"}) || arg[{"-h", "--help"}] ) {
        showHelp();
        return 0;
    }

    if ( arg("--version") || arg["--version"] ) {
        showVersion();
        return 0;
    }

    RuntimeResult result;
    RuntimeConfig config;

    if ( arg({"-l", "--log_path"}) ) {
        std::string path;
        arg("-l") >> path;
        config.log_path = path;
        Log::openFile(path.c_str());
    }

    if ( arg({"-v", "--verbose"}) || arg[{"-v", "--verbose"}] ) {
        log::isDebug();
    }

    // set config
    arg({"-t", "--max_cpu_time"}, 1) >> config.max_cpu_time;
    config.max_cpu_time = config.max_cpu_time * 1000;

    arg({"-s", "--max_stack"}, 16 * 1024) >> config.max_stack;
    config.max_stack = config.max_stack * 1024;

    arg({"-m", "--max_memory"}, 64 * 1024) >> config.max_memory;
    config.max_memory = config.max_memory * 1024;

    arg({"-q", "--max_output_size"}, 1024) >> config.max_output_size;

    arg({"-f", "--max_open_file_number"}, 6)  >> config.max_open_file_number;

    if (arg({"-c", "--exec_path"})) {
        arg({"-c", "--exec_path"}) >> config.exec_path;
    } else {
        fprintf(stderr, "not exec path");
        return 0;
    }

    arg({"-c", "--exec_env"}, "") >> config.exec_env;

    arg({"-c", "--exec_args"}, "") >> config.exec_args;

    arg({"-i", "--input_path"}, "/dev/stdin") >> config.input_path;

    arg({"-o", "--output_path"}, "/dev/stdout") >> config.output_path;

    arg({"-e", "--error_path"}, "/dev/stderr") >> config.error_path;

    arg({"-p", "--scmp_rule"}, "high") >> config.error_path;

    arg({"-u", "--uid"}, 65534) >> config.uid;

    arg({"-g", "--gid"}, 65534) >> config.gid;

    run(config, result);


}

void showVersion() {
    fprintf(stderr, "%s%s%s",
            "SandBox Design for OnlineJudge\n"
            "  Version is ", version, "\n\n");
}

void showHelp() {
    showVersion();
    fprintf(stderr, "%s",
            "Options:\n"
            "  -t  --max_cpu_time              set cpu real time limit(s), default  1 s \n"
            "  -s  --max_stack                 set process stack limit(kb), default  16 * 1024 kb \n"
            "  -m  --max_memory                set memory limit(kb), default  64 * 1024 kb \n"
            "  -q  --max_output_size           set output limit(byte), default  1024 byte \n"
            "  -f  --max_open_file_number      set program open file number limit, default  6 \n"
            "\n"
            "  -c  --exec_path                 set executable file path \n"
            "  -n  --exec_env                  set exec environment, if have more than one args ,use quotes \n"
            "  -a  --exec_args                 set exec arg, if have more than one args, use quotes"
            "\n"
            "  -i  --input_path                set stdin redirect, default /dev/stdin \n"
            "  -o  --output_path               set stdout redirect, default /dev/stdout \n"
            "  -e  --error_path                set stderr redirect, default /dev/stderr \n"
            "\n"
            "  -u  --uid                       set running user id, default 65534\n"
            "  -g  --gid                       set running group id, default 65534\n"
            "  -p  --scmp_rule                 set running seccomp rule name (compile, low, high), default high\n"
            "\n"
            "  -l  --log_path                  set runtime log path, default /dev/stderr \n"
            "  -v  --verbose                   record log in verbose\n"
            "\n"
            "      --version                   get version\n"
            "  -h  --help                      show this message\n"
            "Notice: If there are multiple parameters , please include it in quotation marks just like --exec_args \"-l 1\"\n"
    );

}