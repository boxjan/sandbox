//
// Created by Boxjan on Dec 15, 2018 13:32.
//

#ifndef SANDBOX_LOG_H
#define SANDBOX_LOG_H

#include <cstdio>
#include <cstdarg>
#include <ctime>
#include <sys/file.h>
#include <unistd.h>

enum LEVEL {DEBUG, INFO, WARN, ERROR};
const char LEVEL_STR[][8] = {"DEBUG", "INFO", "WARN", "ERROR"};
const int one_line_max_size = 10240;

static FILE *log_file = nullptr;
static bool is_debug = false;

struct Log {

    static void openFile(const char *file_path) {
        if (log_file != nullptr)
            closeFile();

        log_file = fopen(file_path, "a+");
    }

    static void closeFile() {
        fclose(log_file);
        log_file = nullptr;
    }

    static void debug(const char *format, ...) {
        if ( is_debug ) return;
        va_list args;
        va_start(args, format);
        writeLog(DEBUG, format, args);
        va_end(args);
    }

    static void info(const char *format, ...) {
        va_list args;
        va_start(args, format);
        writeLog(INFO, format, args);
        va_end(args);
    }

    static void warn(const char *format, ...) {
        va_list args;
        va_start(args, format);
        writeLog(WARN, format, args);
        va_end(args);
    }

    static void error(const char *format, ...) {
        va_list args;
        va_start(args, format);
        writeLog(ERROR, format, args);
        va_end(args);
    }

    static void isDebug() {
        is_debug = true;
    }

private:
    static void writeLog(const LEVEL level, const char *format, va_list &args) {
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
        if (log_file == nullptr) {
            fprintf(stderr, "%s", log_str);
            return;
        }

        int log_fd = fileno((FILE *) log_file);
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
};

typedef Log log;
#endif //SANDBOX_LOG_H
