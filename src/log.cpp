//
// Created by Boxjan on Dec 19, 2018 16:38.
//

#include <cstring>
#include <sys/time.h>
#include "log.h"

Log *Log::instance = nullptr;

void Log::build(const char *logPath, bool isDebug) {
    if (instance == nullptr) {
        instance = new Log(logPath, isDebug);
    }
}

Log::Log(const char *log_path_ptr, bool is_debug) {
    if (strlen(log_path_ptr) == 0 || strcmp(log_path_ptr, "stderr") == 0) {
        log_file = nullptr;
    } else {
        strncpy(this->log_path, log_path_ptr, 1024);
        log_file = fopen(log_path_ptr, "a+");
    }
    this->is_debug = is_debug;
}

Log::~Log()  {
    fclose(log_file);
}

Log* Log::getInstance() {
    if (instance == nullptr) {
        build("stderr", false);
    }
    return instance;
}

void Log::closeFile() {
    if (strlen(log_path) == 0 || strcmp(log_path, "stderr") == 0) {
        return;
    }
    fclose(getInstance()->log_file);
    getInstance()->log_file = nullptr;
}

void Log::saveLog(int level, const char *file, int line, const char * function, const char *format, ...) {

    if (level == DEBUG && !getInstance()->is_debug)
        return;

    static char message[one_line_max_size];
    va_list args;
    va_start(args, format);
    vsnprintf(message, one_line_max_size, format, args);
    va_end(args);

    static char timeString[1024];
    struct timeval tv;
    gettimeofday(&tv, nullptr);

    time_t now = tv.tv_sec;
    strftime(timeString, 99, "%Y-%m-%d %H:%M:%S", localtime(&now));

    static char log_str[one_line_max_size];
    snprintf(log_str, one_line_max_size - 1, "%s.%d [%s] [%s] [%s:%d] - %s\n",
            timeString, (int) tv.tv_usec / 100, LEVEL_STR[level], function, file, line, message);

    getInstance()->writeLog(log_str);

}

void Log::writeLog(const char *message) {
    if (getInstance()->log_file == nullptr) {
        getInstance()->writeToStderr(message);
    } else {
        getInstance()->writeToFile(message);
    }

}

void Log::writeToFile(const char *message) {
    int count = (int) strlen(message);

    int log_fd = fileno((FILE *) getInstance()->log_file);
    if (flock(log_fd, LOCK_EX) == 0) {
        if (write(log_fd, message, (size_t) count) < 0) {
            fprintf(stderr, "Can not write log into File: %s, will write into stderr\n", this->log_path);
            this->closeFile();
            writeLog(message);
            return;
        }
        flock(log_fd, LOCK_UN);
    } else {
        fprintf(stderr, "lock file error\n");
        writeLog(message);
    }
}

void Log::writeToStderr(const char *message) {
    fprintf(stderr, "%s", message);
}

