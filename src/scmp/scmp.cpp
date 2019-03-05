//
// Created by Boxjan on Dec 23, 2018 12:31.
//

#include "scmp.h"
#include "../log.h"
#include <string>
#include <cstring>

int load_compile_rule();
int load_gentle_rule(const char *);
int load_strict_rule(const char *);

int load(const char *rule_name, const char *exec_path) {
    if ( strcmp(rule_name, "compile") == 0 ) {
        return load_compile_rule();
    } else if ( strcmp(rule_name, "gentle") == 0 ) {
        return load_gentle_rule(exec_path);
    } else if ( strcmp(rule_name, "strict") == 0 ) {
        return load_strict_rule(exec_path);
    }
   LOG_ERROR("can not load %s seccomp rule", rule_name);
    return -1;
}