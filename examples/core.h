// core.h - Core library header
#ifndef CORE_H
#define CORE_H

#include <string>
#include <stdexcept>

class Core {
public:
    Core(const std::string& name);
    Core(const std::string& name, int value);
    void initialize();
    void initialize(int value);
    std::string getName() const;
    void setName(const std::string& name);
    int getValue() const;
    void setValue(int value);
    bool isInitialized() const;
    void reset();
    bool isValid() const;
    std::string toString() const;
    void process();
    void process(int times);
    void dump() const;
    int compare(const Core& other) const;
    bool equals(const Core& other) const;
    void swap(Core& other);

private:
    std::string name_;
    int value_;
    bool initialized_;
};

#endif