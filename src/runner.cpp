//
// Created by Boxjan on Dec 15, 2018 11:59.
//

#include <unistd.h>
#include <sys/types.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <sys/wait.h>
#include <sched.h>

#include "child.h"
#include "runner.h"
#include "log.h"

int run(const RuntimeConfig &config, RuntimeResult &result) {

    if (getuid() != 0) {
        RUN_EXIT(NOT_RUNNING_BY_ROOT);
    }

    pid_t pid = fork();

    if (pid == 0) {
        child(config);
    }



    return 0;

}