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
    char log_path[1024];

    void writeLog(const char *message);
    void writeToFile(const char *message);
    void writeToStderr(const char *message);
    void closeFile();

    Log(const char * logPath, bool isDebug);
    ~Log();

public:
    static Log *getInstance();

    static void saveLog(int level, const char* file, int line, const char * function, const char * format, ...);
    static void build(const char *logPath, bool isDebug);


};

typedef Log log;


#define LOG_DEBUG(format, x...) log::saveLog(DEBUG, __FILE__, __LINE__, __FUNCTION__, format, ##x);
#define LOG_INFO(format, x...) log::saveLog(INFO, __FILE__, __LINE__, __FUNCTION__, format, _##x);
#define LOG_WARN(format, x...) log::saveLog(WARN, __FILE__, __LINE__, __FUNCTION__, format, ##x);
#define LOG_ERROR(format, x...) log::saveLog(ERROR, __FILE__, __LINE__, __FUNCTION__, format, ##x);

#endif //SANDBOX_LOG_H
