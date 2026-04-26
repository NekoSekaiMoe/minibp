#ifndef STRUTIL_H
#define STRUTIL_H

#include <string>
#include <vector>

namespace strutil {

std::string to_upper(const std::string& str);
std::string to_lower(const std::string& str);
std::string trim(const std::string& str);
std::string trim_left(const std::string& str);
std::string trim_right(const std::string& str);
std::string join(const std::string& separator, const std::vector<std::string>& parts);
std::vector<std::string> split(const std::string& str, const std::string& delimiter);
std::string replace(const std::string& str, const std::string& old_str, const std::string& new_str);
std::string replace_all(const std::string& str, const std::string& old_str, const std::string& new_str);
bool starts_with(const std::string& str, const std::string& prefix);
bool ends_with(const std::string& str, const std::string& suffix);
bool contains(const std::string& str, const std::string& sub);
std::string reverse(const std::string& str);
std::string repeat(const std::string& str, size_t count);
std::string capitalize(const std::string& str);
std::string title_case(const std::string& str);
bool is_numeric(const std::string& str);
bool is_alpha(const std::string& str);
bool is_alphanumeric(const std::string& str);
std::string left_pad(const std::string& str, size_t width, char padding);
std::string right_pad(const std::string& str, size_t width, char padding);

} // namespace strutil

#endif // STRUTIL_H