#ifndef LOGGER_H
#define LOGGER_H

#define LOG_LEVEL_INFO 0
#define LOG_LEVEL_WARN 1
#define LOG_LEVEL_ERROR 2
#define LOG_LEVEL_DEBUG 3

// 设置日志级别
void log_set_level(int level);

// 输出日志
void log_info(const char* format, ...);
void log_warn(const char* format, ...);
void log_error(const char* format, ...);
void log_debug(const char* format, ...);

#endif // LOGGER_H
