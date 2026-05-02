#pragma once

namespace mylib {

class Buffer {
public:
    Buffer();
    ~Buffer();
    void append(const char* data, std::size_t n);

private:
    char* data_;
    std::size_t len_;
};

}  // namespace mylib
