#include "strutil.h"
#include <algorithm>
#include <cctype>
#include <sstream>
#include <vector>

namespace strutil {

std::string to_upper(const std::string& str) {
    std::string result = str;
    std::transform(result.begin(), result.end(), result.begin(),
                   [](unsigned char c) { return std::toupper(c); });
    return result;
}

std::string to_lower(const std::string& str) {
    std::string result = str;
    std::transform(result.begin(), result.end(), result.begin(),
                   [](unsigned char c) { return std::tolower(c); });
    return result;
}

std::string trim(const std::string& str) {
    auto start = std::find_if_not(str.begin(), str.end(),
                                  [](unsigned char c) { return std::isspace(c); });
    auto end = std::find_if_not(str.rbegin(), str.rend(),
                                [](unsigned char c) { return std::isspace(c); }).base();
    return (start < end) ? std::string(start, end) : std::string();
}

std::string trim_left(const std::string& str) {
    auto start = std::find_if_not(str.begin(), str.end(),
                                  [](unsigned char c) { return std::isspace(c); });
    return std::string(start, str.end());
}

std::string trim_right(const std::string& str) {
    auto end = std::find_if_not(str.rbegin(), str.rend(),
                                [](unsigned char c) { return std::isspace(c); }).base();
    return std::string(str.begin(), end);
}

std::string join(const std::string& separator, const std::vector<std::string>& parts) {
    std::ostringstream result;
    for (size_t i = 0; i < parts.size(); ++i) {
        if (i > 0) result << separator;
        result << parts[i];
    }
    return result.str();
}

std::vector<std::string> split(const std::string& str, const std::string& delimiter) {
    std::vector<std::string> result;
    if (str.empty()) return result;
    
    size_t start = 0;
    size_t end = str.find(delimiter);
    
    while (end != std::string::npos) {
        result.push_back(str.substr(start, end - start));
        start = end + delimiter.length();
        end = str.find(delimiter, start);
    }
    result.push_back(str.substr(start));
    return result;
}

std::string replace(const std::string& str, const std::string& old_str, const std::string& new_str) {
    std::string result = str;
    size_t pos = 0;
    while ((pos = result.find(old_str, pos)) != std::string::npos) {
        result.replace(pos, old_str.length(), new_str);
        pos += new_str.length();
    }
    return result;
}

std::string replace_all(const std::string& str, const std::string& old_str, const std::string& new_str) {
    return replace(str, old_str, new_str);
}

bool starts_with(const std::string& str, const std::string& prefix) {
    if (prefix.length() > str.length()) return false;
    return str.compare(0, prefix.length(), prefix) == 0;
}

bool ends_with(const std::string& str, const std::string& suffix) {
    if (suffix.length() > str.length()) return false;
    return str.compare(str.length() - suffix.length(), suffix.length(), suffix) == 0;
}

bool contains(const std::string& str, const std::string& sub) {
    return str.find(sub) != std::string::npos;
}

std::string reverse(const std::string& str) {
    std::string result = str;
    std::reverse(result.begin(), result.end());
    return result;
}

std::string repeat(const std::string& str, size_t count) {
    std::string result;
    for (size_t i = 0; i < count; i++) {
        result += str;
    }
    return result;
}

std::string capitalize(const std::string& str) {
    if (str.empty()) return str;
    std::string result = to_lower(str);
    result[0] = std::toupper(result[0]);
    return result;
}

std::string title_case(const std::string& str) {
    std::string result = str;
    bool capitalize_next = true;
    for (size_t i = 0; i < result.length(); i++) {
        if (capitalize_next && std::isalpha(result[i])) {
            result[i] = std::toupper(result[i]);
            capitalize_next = false;
        } else if (std::isspace(result[i])) {
            capitalize_next = true;
        }
    }
    return result;
}

bool is_numeric(const std::string& str) {
    if (str.empty()) return false;
    size_t start = 0;
    if (str[0] == '-' || str[0] == '+') start = 1;
    for (size_t i = start; i < str.length(); i++) {
        if (!std::isdigit(str[i])) return false;
    }
    return true;
}

bool is_alpha(const std::string& str) {
    if (str.empty()) return false;
    for (char c : str) {
        if (!std::isalpha(c)) return false;
    }
    return true;
}

bool is_alphanumeric(const std::string& str) {
    if (str.empty()) return false;
    for (char c : str) {
        if (!std::isalnum(c)) return false;
    }
    return true;
}

std::string left_pad(const std::string& str, size_t width, char padding) {
    if (str.length() >= width) return str;
    return std::string(width - str.length(), padding) + str;
}

std::string right_pad(const std::string& str, size_t width, char padding) {
    if (str.length() >= width) return str;
    return str + std::string(width - str.length(), padding);
}

} // namespace strutil