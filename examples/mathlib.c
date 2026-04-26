#include "mathlib.h"
#include <stdlib.h>

int gcd(int a, int b) {
    while (b != 0) {
        int temp = b;
        b = a % b;
        a = temp;
    }
    return a;
}

int lcm(int a, int b) {
    if (a == 0 || b == 0) return 0;
    return (a * b) / gcd(a, b);
}

int is_prime(int n) {
    if (n <= 1) return 0;
    if (n <= 3) return 1;
    if (n % 2 == 0 || n % 3 == 0) return 0;
    
    for (int i = 5; i * i <= n; i += 6) {
        if (n % i == 0 || n % (i + 2) == 0) {
            return 0;
        }
    }
    return 1;
}

long long factorial(int n) {
    if (n < 0) return -1;
    if (n == 0 || n == 1) return 1;
    
    long long result = 1;
    for (int i = 2; i <= n; i++) {
        result *= i;
    }
    return result;
}

int power(int base, int exp) {
    if (exp < 0) return 0;
    if (exp == 0) return 1;
    
    int result = 1;
    while (exp > 0) {
        if (exp & 1) result *= base;
        base *= base;
        exp >>= 1;
    }
    return result;
}

double power_double(double base, int exp) {
    if (exp == 0) return 1.0;
    if (exp < 0) {
        base = 1.0 / base;
        exp = -exp;
    }
    double result = base;
    for (int i = 1; i < exp; i++) {
        result *= base;
    }
    return result;
}

double sqrt_approx(double n) {
    if (n < 0) return 0.0;
    if (n == 0) return 0.0;
    
    double guess = n / 2.0;
    for (int i = 0; i < 20; i++) {
        guess = (guess + n / guess) / 2.0;
    }
    return guess;
}

int fibonacci(int n) {
    if (n <= 0) return 0;
    if (n == 1) return 1;
    
    int a = 0, b = 1;
    for (int i = 2; i <= n; i++) {
        int temp = a + b;
        a = b;
        b = temp;
    }
    return b;
}

int fibonacci_rec(int n) {
    if (n <= 0) return 0;
    if (n == 1) return 1;
    return fibonacci_rec(n - 1) + fibonacci_rec(n - 2);
}

int mod_pow(int base, int exp, int mod) {
    int result = 1;
    base = base % mod;
    while (exp > 0) {
        if (exp & 1) result = (result * base) % mod;
        base = (base * base) % mod;
        exp >>= 1;
    }
    return result;
}

int count_digits(int n) {
    if (n == 0) return 1;
    int count = 0;
    if (n < 0) n = -n;
    while (n > 0) {
        count++;
        n /= 10;
    }
    return count;
}

int reverse_int(int n) {
    int reversed = 0;
    while (n != 0) {
        reversed = reversed * 10 + n % 10;
        n /= 10;
    }
    return reversed;
}

int is_palindrome_num(int n) {
    if (n < 0) return 0;
    int original = n;
    int reversed = 0;
    while (n > 0) {
        reversed = reversed * 10 + n % 10;
        n /= 10;
    }
    return original == reversed;
}

int sum_of_digits(int n) {
    int sum = 0;
    if (n < 0) n = -n;
    while (n > 0) {
        sum += n % 10;
        n /= 10;
    }
    return sum;
}

int next_prime(int n) {
    if (n < 2) return 2;
    int candidate = n + 1;
    while (!is_prime(candidate)) {
        candidate++;
    }
    return candidate;
}

int nth_prime(int n) {
    if (n <= 0) return 0;
    int count = 0;
    int num = 2;
    while (count < n) {
        if (is_prime(num)) {
            count++;
            if (count == n) return num;
        }
        num++;
    }
    return 0;
}