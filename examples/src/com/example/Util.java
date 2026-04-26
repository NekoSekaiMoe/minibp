// Util.java - Utility class
package com.example;

public class Util {
    public static String greet(String name) {
        return "Hello, " + name + "!";
    }
    
    public static int add(int a, int b) {
        return a + b;
    }
    
    public static int subtract(int a, int b) {
        return a - b;
    }
    
    public static int multiply(int a, int b) {
        return a * b;
    }
    
    public static int divide(int a, int b) {
        if (b == 0) {
            throw new ArithmeticException("Division by zero");
        }
        return a / b;
    }
    
    public static int modulo(int a, int b) {
        if (b == 0) {
            throw new ArithmeticException("Modulo by zero");
        }
        return a % b;
    }
    
    public static int max(int a, int b) {
        return Math.max(a, b);
    }
    
    public static int min(int a, int b) {
        return Math.min(a, b);
    }
    
    public static int abs(int n) {
        return Math.abs(n);
    }
    
    public static int clamp(int value, int minVal, int maxVal) {
        return Math.max(minVal, Math.min(maxVal, value));
    }
    
    public static boolean isEven(int n) {
        return n % 2 == 0;
    }
    
    public static boolean isOdd(int n) {
        return n % 2 != 0;
    }
    
    public static int sign(int n) {
        return Integer.signum(n);
    }
    
    public static boolean isPrime(int n) {
        if (n <= 1) return false;
        if (n <= 3) return true;
        if (n % 2 == 0 || n % 3 == 0) return false;
        
        for (int i = 5; i * i <= n; i += 6) {
            if (n % i == 0 || n % (i + 2) == 0) {
                return false;
            }
        }
        return true;
    }
    
    public static long factorial(int n) {
        if (n < 0) throw new IllegalArgumentException("Negative number");
        long result = 1;
        for (int i = 2; i <= n; i++) {
            result *= i;
        }
        return result;
    }
    
    public static int fibonacci(int n) {
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
    
    public static int gcd(int a, int b) {
        while (b != 0) {
            int temp = b;
            b = a % b;
            a = temp;
        }
        return a;
    }
    
    public static int lcm(int a, int b) {
        if (a == 0 || b == 0) return 0;
        return (a * b) / gcd(a, b);
    }
    
    public static int power(int base, int exp) {
        if (exp < 0) return 0;
        if (exp == 0) return 1;
        
        int result = 1;
        while (exp > 0) {
            if ((exp & 1) == 1) result *= base;
            base *= base;
            exp >>= 1;
        }
        return result;
    }
    
    public static int sumOfDigits(int n) {
        int sum = 0;
        n = Math.abs(n);
        while (n > 0) {
            sum += n % 10;
            n /= 10;
        }
        return sum;
    }
    
    public static int reverse(int n) {
        int reversed = 0;
        while (n != 0) {
            reversed = reversed * 10 + n % 10;
            n /= 10;
        }
        return reversed;
    }
    
    public static boolean isPalindrome(int n) {
        return n == reverse(Math.abs(n));
    }
    
    public static int countDigits(int n) {
        if (n == 0) return 1;
        int count = 0;
        n = Math.abs(n);
        while (n > 0) {
            count++;
            n /= 10;
        }
        return count;
    }
    
    public static double max(double a, double b) {
        return Math.max(a, b);
    }
    
    public static double min(double a, double b) {
        return Math.min(a, b);
    }
    
    public static double abs(double n) {
        return Math.abs(n);
    }
    
    public static double sqrt(double n) {
        return Math.sqrt(n);
    }
    
    public static double pow(double base, double exp) {
        return Math.pow(base, exp);
    }
}