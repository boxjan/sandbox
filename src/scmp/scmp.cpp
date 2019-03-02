//
// Created by Boxjan on Dec 23, 2018 12:31.
//

#include "scmp.h"
#include "../log.h"
#include <string>

int load_compile_rule();
int load_gentle_rule(const std::string &);
int load_strict_rule(const std::string &);

int load(const std::string &rule_name, const std::string &exec_path) {
    if (rule_name == "compile") {
        return load_compile_rule();
    } else if (rule_name == "gentle") {
        return load_gentle_rule(exec_path);
    } else if (rule_name == "strict") {
        return load_strict_rule(exec_path);
    }
   LOG_ERROR("can not load %s seccomp rule", rule_name.c_str());
    return -1;
}