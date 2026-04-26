#ifndef UTIL_H
#define UTIL_H

#include <stddef.h>

void print_hello(void);
int add(int a, int b);
int subtract(int a, int b);
int multiply(int a, int b);
int divide(int a, int b);
int modulo(int a, int b);
int max_int(int a, int b);
int min_int(int a, int b);
int abs_int(int n);
int clamp_int(int value, int min_val, int max_val);
void swap_int(int* a, int* b);
int is_even(int n);
int is_odd(int n);
int is_negative(int n);
int is_positive(int n);
int sign_int(int n);
long long add_long(long long a, long long b);
long long subtract_long(long long a, long long b);
long long multiply_long(long long a, long long b);
long long divide_long(long long a, long long b);
double add_double(double a, double b);
double subtract_double(double a, double b);
double multiply_double(double a, double b);
double divide_double(double a, double b);
double max_double(double a, double b);
double min_double(double a, double b);
double abs_double(double n);

#endif