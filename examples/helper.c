// helper.c - Helper functions
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <stdarg.h>

void log_message(const char* msg) {
    printf("[LOG] %s\n", msg);
}

void log_message_with_time(const char* msg) {
    time_t now = time(NULL);
    struct tm* tm_info = localtime(&now);
    char timestamp[20];
    strftime(timestamp, sizeof(timestamp), "%Y-%m-%d %H:%M:%S", tm_info);
    printf("[LOG] [%s] %s\n", timestamp, msg);
}

void log_format(const char* format, ...) {
    va_list args;
    va_start(args, format);
    printf("[LOG] ");
    vprintf(format, args);
    printf("\n");
    va_end(args);
}

int is_valid_string(const char* str) {
    return (str != NULL && str[0] != '\0');
}

int string_length(const char* str) {
    if (str == NULL) return 0;
    return (int)strlen(str);
}

void string_copy(char* dest, const char* src, size_t dest_size) {
    if (dest == NULL || src == NULL || dest_size == 0) return;
    size_t i = 0;
    while (i < dest_size - 1 && src[i] != '\0') {
        dest[i] = src[i];
        i++;
    }
    dest[i] = '\0';
}

int string_compare(const char* s1, const char* s2) {
    if (s1 == NULL && s2 == NULL) return 0;
    if (s1 == NULL) return -1;
    if (s2 == NULL) return 1;
    return strcmp(s1, s2);
}

int string_starts_with(const char* str, const char* prefix) {
    if (str == NULL || prefix == NULL) return 0;
    size_t prefix_len = strlen(prefix);
    if (strlen(str) < prefix_len) return 0;
    return strncmp(str, prefix, prefix_len) == 0;
}

int string_ends_with(const char* str, const char* suffix) {
    if (str == NULL || suffix == NULL) return 0;
    size_t str_len = strlen(str);
    size_t suffix_len = strlen(suffix);
    if (str_len < suffix_len) return 0;
    return strcmp(str + str_len - suffix_len, suffix) == 0;
}

void trim_whitespace(char* str) {
    if (str == NULL) return;
    size_t len = strlen(str);
    size_t start = 0;
    while (start < len && (str[start] == ' ' || str[start] == '\t' || str[start] == '\n' || str[start] == '\r')) {
        start++;
    }
    size_t end = len;
    while (end > start && (str[end - 1] == ' ' || str[end - 1] == '\t' || str[end - 1] == '\n' || str[end - 1] == '\r')) {
        end--;
    }
    size_t new_len = end - start;
    if (start > 0) {
        for (size_t i = 0; i < new_len; i++) {
            str[i] = str[start + i];
        }
    }
    str[new_len] = '\0';
}

int to_upper_case(char* str) {
    if (str == NULL) return 0;
    int changed = 0;
    for (int i = 0; str[i] != '\0'; i++) {
        if (str[i] >= 'a' && str[i] <= 'z') {
            str[i] = str[i] - 32;
            changed = 1;
        }
    }
    return changed;
}

int to_lower_case(char* str) {
    if (str == NULL) return 0;
    int changed = 0;
    for (int i = 0; str[i] != '\0'; i++) {
        if (str[i] >= 'A' && str[i] <= 'Z') {
            str[i] = str[i] + 32;
            changed = 1;
        }
    }
    return changed;
}

int is_digit_char(char c) {
    return (c >= '0' && c <= '9');
}

int is_alpha_char(char c) {
    return ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'));
}

int is_alphanumeric_char(char c) {
    return is_digit_char(c) || is_alpha_char(c);
}

int is_whitespace_char(char c) {
    return (c == ' ' || c == '\t' || c == '\n' || c == '\r');
}

int count_words(const char* str) {
    if (str == NULL || str[0] == '\0') return 0;
    int count = 0;
    int in_word = 0;
    for (int i = 0; str[i] != '\0'; i++) {
        if (is_whitespace_char(str[i])) {
            in_word = 0;
        } else if (!in_word) {
            in_word = 1;
            count++;
        }
    }
    return count;
}

char* get_extension(const char* filename) {
    if (filename == NULL) return NULL;
    const char* dot = strrchr(filename, '.');
    if (dot == NULL || dot == filename) return "";
    return (char*)(dot + 1);
}

char* get_basename(const char* path) {
    if (path == NULL) return NULL;
    const char* sep = strrchr(path, '/');
    if (sep == NULL) return (char*)path;
    return (char*)(sep + 1);
}

void get_timestamp(char* buffer, size_t size) {
    if (buffer == NULL || size == 0) return;
    time_t now = time(NULL);
    struct tm* tm_info = localtime(&now);
    strftime(buffer, size, "%Y-%m-%d %H:%M:%S", tm_info);
}