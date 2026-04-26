#ifndef ARRAYUTIL_H
#define ARRAYUTIL_H

#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

long long array_sum(const int* arr, size_t size);
double array_average(const int* arr, size_t size);
int array_max(const int* arr, size_t size);
int array_min(const int* arr, size_t size);
void array_reverse(int* arr, size_t size);
int array_find(const int* arr, size_t size, int value);
int array_contains(const int* arr, size_t size, int value);
size_t array_count(const int* arr, size_t size, int value);
void array_sort(int* arr, size_t size);
int array_binary_search(const int* arr, size_t size, int value);
void array_copy(const int* src, size_t src_size, int* dest, size_t dest_size);
void array_fill(int* arr, size_t size, int value);
int array_sum_long(const long* arr, size_t size);
double array_average_double(const double* arr, size_t size);
double array_max_double(const double* arr, size_t size);
double array_min_double(const double* arr, size_t size);
void array_shuffle(int* arr, size_t size);
int array_is_sorted(const int* arr, size_t size);
int array_is_sorted_desc(const int* arr, size_t size);

#ifdef __cplusplus
}
#endif

#endif // ARRAYUTIL_H