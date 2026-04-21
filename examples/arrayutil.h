#ifndef ARRAYUTIL_H
#define ARRAYUTIL_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// 计算整数数组的和
long long array_sum(const int* arr, size_t size);

// 计算整数数组的平均值
double array_average(const int* arr, size_t size);

// 查找数组中的最大值
int array_max(const int* arr, size_t size);

// 查找数组中的最小值
int array_min(const int* arr, size_t size);

// 反转数组
void array_reverse(int* arr, size_t size);

#ifdef __cplusplus
}
#endif

#endif // ARRAYUTIL_H
