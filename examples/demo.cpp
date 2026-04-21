#include <iostream>
#include <string>

int main() {
    std::cout << "Hello from C++!" << std::endl;
    std::cout << "This is a simple C++ demo program." << std::endl;
    
    // Simple string operations
    std::string greeting = "Hello";
    std::string world = "World";
    
    std::cout << greeting << ", " << world << "!" << std::endl;
    
    // Simple calculation
    int a = 10, b = 20;
    std::cout << a << " + " << b << " = " << (a + b) << std::endl;
    std::cout << a << " * " << b << " = " << (a * b) << std::endl;
    
    return 0;
}
