// core.cpp - Core library
#include "core.h"
#include <iostream>
#include <algorithm>
#include <stdexcept>

Core::Core(const std::string& name) : name_(name), initialized_(false) {}

Core::Core(const std::string& name, int value) : name_(name), value_(value), initialized_(false) {}

void Core::initialize() {
    std::cout << "Initializing core: " << name_ << std::endl;
    initialized_ = true;
}

void Core::initialize(int value) {
    value_ = value;
    initialize();
}

std::string Core::getName() const {
    return name_;
}

void Core::setName(const std::string& name) {
    name_ = name;
}

int Core::getValue() const {
    return value_;
}

void Core::setValue(int value) {
    value_ = value;
}

bool Core::isInitialized() const {
    return initialized_;
}

void Core::reset() {
    name_ = "";
    value_ = 0;
    initialized_ = false;
}

bool Core::isValid() const {
    return !name_.empty();
}

std::string Core::toString() const {
    return "Core[name=" + name_ + ", value=" + std::to_string(value_) + "]";
}

void Core::process() {
    if (!initialized_) {
        throw std::runtime_error("Core not initialized: " + name_);
    }
    std::cout << "Processing: " << name_ << " (value=" << value_ << ")" << std::endl;
}

void Core::process(int times) {
    for (int i = 0; i < times; i++) {
        process();
    }
}

void Core::dump() const {
    std::cout << toString() << std::endl;
}

int Core::compare(const Core& other) const {
    if (value_ < other.value_) return -1;
    if (value_ > other.value_) return 1;
    return name_.compare(other.name_);
}

bool Core::equals(const Core& other) const {
    return name_ == other.name_ && value_ == other.value_;
}

void Core::swap(Core& other) {
    std::swap(name_, other.name_);
    std::swap(value_, other.value_);
    std::swap(initialized_, other.initialized_);
}