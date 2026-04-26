// util.c - Utility functions
#include "util.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void print_hello() {
    printf("Hello from util!\n");
}

int add(int a, int b) {
    return a + b;
}

int subtract(int a, int b) {
    return a - b;
}

int multiply(int a, int b) {
    return a * b;
}

int divide(int a, int b) {
    if (b == 0) {
        return 0;
    }
    return a / b;
}

int modulo(int a, int b) {
    if (b == 0) {
        return 0;
    }
    return a % b;
}

int max_int(int a, int b) {
    return (a > b) ? a : b;
}

int min_int(int a, int b) {
    return (a < b) ? a : b;
}

int abs_int(int n) {
    return (n < 0) ? -n : n;
}

int clamp_int(int value, int min_val, int max_val) {
    if (value < min_val) return min_val;
    if (value > max_val) return max_val;
    return value;
}

void swap_int(int* a, int* b) {
    if (a == NULL || b == NULL) return;
    int temp = *a;
    *a = *b;
    *b = temp;
}

int is_even(int n) {
    return (n % 2 == 0) ? 1 : 0;
}

int is_odd(int n) {
    return (n % 2 != 0) ? 1 : 0;
}

int is_negative(int n) {
    return (n < 0) ? 1 : 0;
}

int is_positive(int n) {
    return (n > 0) ? 1 : 0;
}

int sign_int(int n) {
    if (n > 0) return 1;
    if (n < 0) return -1;
    return 0;
}

long long add_long(long long a, long long b) {
    return a + b;
}

long long subtract_long(long long a, long long b) {
    return a - b;
}

long long multiply_long(long long a, long long b) {
    return a * b;
}

long long divide_long(long long a, long long b) {
    if (b == 0) return 0;
    return a / b;
}

double add_double(double a, double b) {
    return a + b;
}

double subtract_double(double a, double b) {
    return a - b;
}

double multiply_double(double a, double b) {
    return a * b;
}

double divide_double(double a, double b) {
    if (b == 0.0) return 0.0;
    return a / b;
}

double max_double(double a, double b) {
    return (a > b) ? a : b;
}

double min_double(double a, double b) {
    return (a < b) ? a : b;
}

double abs_double(double n) {
    return (n < 0.0) ? -n : n;
}