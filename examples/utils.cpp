// utils.cpp - Utility functions
#include <algorithm>
#include <string>
#include <cctype>
#include <sstream>
#include <vector>
#include <stdexcept>

std::string to_upper(const std::string& s) {
    std::string result = s;
    std::transform(result.begin(), result.end(), result.begin(), ::toupper);
    return result;
}

std::string to_lower(const std::string& s) {
    std::string result = s;
    std::transform(result.begin(), result.end(), result.begin(), ::tolower);
    return result;
}

std::string trim(const std::string& s) {
    auto start = std::find_if_not(s.begin(), s.end(),
                                 [](unsigned char c) { return std::isspace(c); });
    auto end = std::find_if_not(s.rbegin(), s.rend(),
                               [](unsigned char c) { return std::isspace(c); }).base();
    return (start < end) ? std::string(start, end) : std::string();
}

int add_int(int a, int b) {
    return a + b;
}

int subtract_int(int a, int b) {
    return a - b;
}

int multiply_int(int a, int b) {
    return a * b;
}

int divide_int(int a, int b) {
    if (b == 0) return 0;
    return a / b;
}

int max_int(int a, int b) {
    return std::max(a, b);
}

int min_int(int a, int b) {
    return std::min(a, b);
}

int abs_int(int n) {
    return std::abs(n);
}

bool is_even(int n) {
    return n % 2 == 0;
}

bool is_odd(int n) {
    return n % 2 != 0;
}

int clamp_int(int value, int min_val, int max_val) {
    return std::max(min_val, std::min(max_val, value));
}

std::string join(const std::string& separator, const std::vector<std::string>& parts) {
    std::ostringstream result;
    for (size_t i = 0; i < parts.size(); ++i) {
        if (i > 0) result << separator;
        result << parts[i];
    }
    return result.str();
}

std::vector<std::string> split(const std::string& s, const std::string& delimiter) {
    std::vector<std::string> result;
    if (s.empty()) return result;
    
    size_t start = 0;
    size_t end = s.find(delimiter);
    
    while (end != std::string::npos) {
        result.push_back(s.substr(start, end - start));
        start = end + delimiter.length();
        end = s.find(delimiter, start);
    }
    result.push_back(s.substr(start));
    return result;
}

std::string replace(const std::string& s, const std::string& old_str, const std::string& new_str) {
    std::string result = s;
    size_t pos = 0;
    while ((pos = result.find(old_str, pos)) != std::string::npos) {
        result.replace(pos, old_str.length(), new_str);
        pos += new_str.length();
    }
    return result;
}

bool starts_with(const std::string& s, const std::string& prefix) {
    if (prefix.length() > s.length()) return false;
    return s.compare(0, prefix.length(), prefix) == 0;
}

bool ends_with(const std::string& s, const std::string& suffix) {
    if (suffix.length() > s.length()) return false;
    return s.compare(s.length() - suffix.length(), suffix.length(), suffix) == 0;
}

bool contains(const std::string& s, const std::string& sub) {
    return s.find(sub) != std::string::npos;
}

std::string reverse(const std::string& s) {
    std::string result = s;
    std::reverse(result.begin(), result.end());
    return result;
}

std::string repeat(const std::string& s, size_t count) {
    std::string result;
    for (size_t i = 0; i < count; i++) {
        result += s;
    }
    return result;
}