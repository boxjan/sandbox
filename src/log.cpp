//
// Created by Boxjan on Dec 19, 2018 16:38.
//

#include "log.h"

Log *Log::instance = nullptr;

Log::Log()  {
    log_file = nullptr;
    is_debug = false;
}

Log::~Log()  {
    closeFile();
}

Log* Log::getInstance() {
    if (instance == nullptr) {
        instance = new Log();
    }
    return instance;
}

void Log::isDebug()  {
    getInstance()->is_debug = true;
}

void Log::openFile(const char *file_path) {
    if (getInstance()->log_file != nullptr)
        closeFile();

    getInstance()->log_file = fopen(file_path, "a+");
}

void Log::closeFile() {
    fclose(getInstance()->log_file);
    getInstance()->log_file = nullptr;
}

void Log::debug(const char *format, ...) {
    va_list args;
    va_start(args, format);
    writeLog(DEBUG, format, args);
    va_end(args);
}

void Log::info(const char *format, ...) {
    va_list args;
    va_start(args, format);
    writeLog(INFO, format, args);
    va_end(args);
}

void Log::warn(const char *format, ...) {
    va_list args;
    va_start(args, format);
    writeLog(WARN, format, args);
    va_end(args);
}

void Log::error(const char *format, ...) {
    va_list args;
    va_start(args, format);
    writeLog(ERROR, format, args);
    va_end(args);
}

void Log::writeLog(const LEVEL level, const char *format, va_list &args) {

    if (level == DEBUG && !getInstance()->is_debug) return;

    // format time
    static char datetime[64];
    static time_t now;
    now = time(nullptr);
    strftime(datetime, 63, "%Y-%m-%d %H:%M:%S", localtime(&now));

    //
    static char message[one_line_max_size];
    vsnprintf(message, one_line_max_size, format, args);

    // get one record
    static char log_str[one_line_max_size];
    int count = snprintf(log_str, one_line_max_size, "%s [%s] - %s\n", datetime, LEVEL_STR[level], message);

    // file
    if (getInstance()->log_file == nullptr) {
        fprintf(stderr, "%s", log_str);
        return;
    }

    int log_fd = fileno((FILE *) getInstance()->log_file);
    if (flock(log_fd, LOCK_EX) == 0) {
        if (write(log_fd, log_str, (size_t) count) < 0) {
            fprintf(stderr, "write error\n");
            fprintf(stderr, "%s", log_str);
            return;
        }
        flock(log_fd, LOCK_UN);
    } else {
        fprintf(stderr, "lock file error\n");
        fprintf(stderr, "%s", log_str);
    }
}

