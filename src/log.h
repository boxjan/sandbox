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

class Log {
private:
    FILE *log_file;
    bool is_debug;
    static Log *instance;

    static void writeLog(const LEVEL level, const char *format, va_list &args);
    Log();
    ~Log();

public:
    static Log *getInstance();

    static void openFile(const char *file_path);
    static void closeFile();

    static void debug(const char *format, ...);
    static void info(const char *format, ...);
    static void warn(const char *format, ...);
    static void error(const char *format, ...);

    static void isDebug();


};

typedef Log log;
#endif //SANDBOX_LOG_H
