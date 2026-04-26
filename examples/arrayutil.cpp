#include "arrayutil.h"
#include <limits.h>
#include <string.h>
#include <stdlib.h>

long long array_sum(const int* arr, size_t size) {
    long long sum = 0;
    for (size_t i = 0; i < size; i++) {
        sum += arr[i];
    }
    return sum;
}

double array_average(const int* arr, size_t size) {
    if (size == 0) return 0.0;
    return (double)array_sum(arr, size) / (double)size;
}

int array_max(const int* arr, size_t size) {
    if (size == 0) return 0;
    int max = arr[0];
    for (size_t i = 1; i < size; i++) {
        if (arr[i] > max) max = arr[i];
    }
    return max;
}

int array_min(const int* arr, size_t size) {
    if (size == 0) return 0;
    int min = arr[0];
    for (size_t i = 1; i < size; i++) {
        if (arr[i] < min) min = arr[i];
    }
    return min;
}

void array_reverse(int* arr, size_t size) {
    for (size_t i = 0; i < size / 2; i++) {
        int temp = arr[i];
        arr[i] = arr[size - 1 - i];
        arr[size - 1 - i] = temp;
    }
}

int array_find(const int* arr, size_t size, int value) {
    for (size_t i = 0; i < size; i++) {
        if (arr[i] == value) return (int)i;
    }
    return -1;
}

int array_contains(const int* arr, size_t size, int value) {
    return array_find(arr, size, value) >= 0;
}

size_t array_count(const int* arr, size_t size, int value) {
    size_t count = 0;
    for (size_t i = 0; i < size; i++) {
        if (arr[i] == value) count++;
    }
    return count;
}

void array_sort(int* arr, size_t size) {
    for (size_t i = 0; i < size; i++) {
        for (size_t j = i + 1; j < size; j++) {
            if (arr[i] > arr[j]) {
                int temp = arr[i];
                arr[i] = arr[j];
                arr[j] = temp;
            }
        }
    }
}

int array_binary_search(const int* arr, size_t size, int value) {
    size_t left = 0;
    size_t right = size;
    
    while (left < right) {
        size_t mid = left + (right - left) / 2;
        if (arr[mid] == value) return (int)mid;
        if (arr[mid] < value) {
            left = mid + 1;
        } else {
            right = mid;
        }
    }
    return -1;
}

void array_copy(const int* src, size_t src_size, int* dest, size_t dest_size) {
    size_t min_size = src_size < dest_size ? src_size : dest_size;
    for (size_t i = 0; i < min_size; i++) {
        dest[i] = src[i];
    }
}

void array_fill(int* arr, size_t size, int value) {
    for (size_t i = 0; i < size; i++) {
        arr[i] = value;
    }
}

int array_sum_long(const long* arr, size_t size) {
    long sum = 0;
    for (size_t i = 0; i < size; i++) {
        sum += arr[i];
    }
    return (int)sum;
}

double array_average_double(const double* arr, size_t size) {
    if (size == 0) return 0.0;
    double sum = 0.0;
    for (size_t i = 0; i < size; i++) {
        sum += arr[i];
    }
    return sum / (double)size;
}

double array_max_double(const double* arr, size_t size) {
    if (size == 0) return 0.0;
    double max = arr[0];
    for (size_t i = 1; i < size; i++) {
        if (arr[i] > max) max = arr[i];
    }
    return max;
}

double array_min_double(const double* arr, size_t size) {
    if (size == 0) return 0.0;
    double min = arr[0];
    for (size_t i = 1; i < size; i++) {
        if (arr[i] < min) min = arr[i];
    }
    return min;
}

void array_shuffle(int* arr, size_t size) {
    for (size_t i = size - 1; i > 0; i--) {
        size_t j = (size_t)rand() % (i + 1);
        int temp = arr[i];
        arr[i] = arr[j];
        arr[j] = temp;
    }
}

int array_is_sorted(const int* arr, size_t size) {
    for (size_t i = 1; i < size; i++) {
        if (arr[i] < arr[i - 1]) return 0;
    }
    return 1;
}

int array_is_sorted_desc(const int* arr, size_t size) {
    for (size_t i = 1; i < size; i++) {
        if (arr[i] > arr[i - 1]) return 0;
    }
    return 1;
}