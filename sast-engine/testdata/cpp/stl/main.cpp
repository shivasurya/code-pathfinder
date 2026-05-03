/*
 * Smoke fixture for C++ STL resolution (PR-02).
 *
 * The calls exercise the three dispatch paths the C++ resolver
 * extends in PR-02:
 *   1. Class methods on stdlib types         (vec.push_back, vec.size)
 *   2. Free functions in std namespace       (std::move, std::swap)
 *   3. C-shape stdlib calls from C++ code    (printf, malloc)
 *
 * The user-defined Echo class verifies project resolution still
 * works alongside stdlib resolution.
 */

#include <cstdio>
#include <cstdlib>
#include <string>
#include <utility>
#include <vector>

class Echo {
  public:
    explicit Echo(std::string prefix) : prefix_(std::move(prefix)) {}
    void say(const std::string &msg) const {
        std::printf("%s: %s\n", prefix_.c_str(), msg.c_str());
    }

  private:
    std::string prefix_;
};

int main() {
    std::vector<int> nums;
    nums.push_back(1);
    nums.push_back(2);
    nums.push_back(3);

    int a = 10, b = 20;
    std::swap(a, b);

    Echo echo("info");
    echo.say("hello, stl");

    void *raw = std::malloc(16);
    if (raw != nullptr) {
        std::free(raw);
    }

    return static_cast<int>(nums.size());
}
