#include "arrayutil.h"
#include <limits.h>

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
