#ifndef MATHLIB_H
#define MATHLIB_H

#ifdef __cplusplus
extern "C" {
#endif

// 计算两个整数的最大公约数
int gcd(int a, int b);

// 计算两个整数的最小公倍数
int lcm(int a, int b);

// 判断一个数是否为质数
int is_prime(int n);

// 计算阶乘
long long factorial(int n);

#ifdef __cplusplus
}
#endif

#endif // MATHLIB_H
