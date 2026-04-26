#ifndef LOGGER_H
#define LOGGER_H

#include <stdio.h>

#define LOG_LEVEL_DEBUG 0
#define LOG_LEVEL_INFO  1
#define LOG_LEVEL_WARN  2
#define LOG_LEVEL_ERROR 3

void log_set_level(int level);
void log_set_file(FILE* file);
void log_enable_colors(int enable);
void log_info(const char* format, ...);
void log_warn(const char* format, ...);
void log_error(const char* format, ...);
void log_debug(const char* format, ...);
void log_fatal(const char* format, ...);
int log_get_level(void);
void log_close(void);

#endif // LOGGER_H