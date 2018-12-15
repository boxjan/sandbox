//
// Created by hzj on 10:00 Dec 15, 2018
//
#include <iostream>
#include <string>

#include "argh.h"
#include "runner.h"
#include "log.h"

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

    if ( arg({"-l", "--log_path"}) ) {
        std::string path;
        arg("-l") >> path;
        Log::openFile(path.c_str());
    }

    RuntimeResult result = {0};
    RuntimeConfig config = {0};

    log::debug("aaa");

}

void showVersion() {
    fprintf(stderr, "%s%s%s",
            "SandBox Design for OnlineJudge\n"
            "  Version is ", version, "\n\n");
}

void showHelp() {
    showVersion();
    fprintf(stderr, "%s",
            "  -t  --max_cpu_time              set cpu real time limit(s), default  1 s \n"
            "  -s  --max_stack                 set process stack limit(kb), default  32 * 1024 kb \n"
            "  -m  --max_memory                set memory limit(kb), default  128 * 1024 kb \n"
            "  -a  --max_output_size           set output limit(byte), default  1024 byte \n"
            "  -p  --max_process_number        set process number limit, default  1 \n"
            "  -f  --max_open_file_number      set program open file number limit, default  16 \n"
            "\n"
            "  -c  --exec_path                 set executable file path \n"
            "  -n  --env                       set exec environment, if have more than one args ,use quotes \n"
            "  -a  --exec_args                 set exec arg, if have more than one args, use quotes"
            "\n"
            "  -i  --input_path                set stdin redirect, default /dev/stdin \n"
            "  -o  --output_path               set stdout redirect, default /dev/stdout \n"
            "  -e  --error_path                set stderr redirect, default /dev/stderr \n"
            "\n"
            "  -u  --uid                       set running user id, default\n"
            "  -g  --gid                       set running group id\n"
            "\n"
            "  -l  --log_path                  set runtime log path, default /dev/stderr \n"
            "  -v  --verbose                   record log in verbose\n"
            "\n"
            "      --version                   get version\n"
            "  -h  --help                      show this message\n"
            "  Notice: If there are multiple parameters , please include it in quotation marks just like --exec_args \"-l 1\"\n"
    );

}