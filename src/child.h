//
// Created by Boxjan on Dec 16, 2018 15:30.
//

#ifndef SANDBOX_CHILD_H
#define SANDBOX_CHILD_H

#include "runner.h"

enum CHILD_EXIT_CODE {
    CPU_LIMIT_FAIL,
    MEMORY_LIMIT_FAIL,
    STACK_LIMIT_FAIL,
    OUTPUT_LIMIT_FAIL,
    OPEN_FILE_COUNT_LIMIT_FAIL,
    OPEN_INPUT_FILE_FAIL,
    MOUNT_INPUT_FILE_FAIL,
    OPEN_OUTPUT_FILE_FAIL,
    MOUNT_OUTPUT_FILE_FAIL,
    OPEN_ERROR_FILE_FAIL,
    MOUNT_ERROR_FILE_FAIL,
    SET_UID_FAIL,
    SET_GID_FAIL,
};

const char CHILD_EXIT_REASON[][32] = {
        "SETTING CPU LIMIT FAIL",
        "SETTING MEMORY LIMIT FAIL",
        "SETTING STACK LIMIT FAIL",
        "SETTING OUTPUT LIMIT FAIL",
        "SETTING OPEN FILE LIMIT FAIL",
        "OPEN INPUT FILE FAIL",
        "MOUNT INPUT FILE FAIL",
        "OPEN OUTPUT FILE FAIL",
        "MOUNT OUTPUT FILE FAIL",
        "OPEN ERROR FILE FAIL",
        "MOUNT ERROR FILE FAIL",
        "SETTING USER ID FAIL",
        "SETTING GROUP ID FAIL"
};     //12345678123456781234567812345678

void child(const RuntimeConfig &);

#define CHILD_EXIT(code) { \
    log::error("child process exit because: %s", CHILD_EXIT_REASON[code]); \
    close_file(IN_FILE); \
    close_file(OUT_FILE); \
    close_file(ERR_FILE); \
    exit(CHILD_FAIL); \
}


#endif //SANDBOX_CHILD_H
