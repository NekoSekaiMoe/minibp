#ifndef MATHLIB_H
#define MATHLIB_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

int gcd(int a, int b);
int lcm(int a, int b);
int is_prime(int n);
long long factorial(int n);
int power(int base, int exp);
double power_double(double base, int exp);
double sqrt_approx(double n);
int fibonacci(int n);
int fibonacci_rec(int n);
int mod_pow(int base, int exp, int mod);
int count_digits(int n);
int reverse_int(int n);
int is_palindrome_num(int n);
int sum_of_digits(int n);
int next_prime(int n);
int nth_prime(int n);

#ifdef __cplusplus
}
#endif

#endif // MATHLIB_H