#include <iostream>
#include <stdexcept>
#include "buffer.hpp"

namespace mylib {

class Animal {
public:
    virtual void speak() = 0;
    virtual ~Animal();
};

class Dog : public Animal {
public:
    void speak() override;
    int age;
private:
    void bark();
};

void Dog::speak() {
    std::cout << "woof" << std::endl;
}

void Dog::bark() {
    speak();
}

Dog::~Animal() {
}

template<typename T>
T identity(T v) {
    return v;
}

void process(const std::string& msg) {
    try {
        if (msg.empty()) {
            throw std::runtime_error("empty");
        }
        identity<int>(42);
    } catch (const std::exception& e) {
        std::cerr << e.what();
    }
}

}  // namespace mylib

namespace {
int hidden_counter = 0;
}

struct Point {
    int x;
    int y;
};

enum class Color { Red, Green, Blue };

typedef unsigned long size_alias;

int main() {
    mylib::Dog d;
    d.speak();
    return 0;
}
