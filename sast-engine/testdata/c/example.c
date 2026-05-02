#include <stdio.h>
#include <stdlib.h>
#include "buffer.h"

typedef unsigned long size_t_alias;
typedef struct {
    int x;
    int y;
} Point;

struct Buffer {
    char* data;
    size_t_alias len;
    int capacity;
};

enum Color {
    RED = 0,
    GREEN,
    BLUE = 5
};

static const float pi = 3.14f;
char* global_buf;
int a = 1, b = 2, c;

int add(int a, int b);

static inline int fast(int x) {
    return x;
}

int add(int a, int b) {
    int result = a + b;
    return result;
}

void process(struct Buffer* buf, size_t_alias n) {
    if (buf == NULL) {
        return;
    }
    char* tmp = malloc(n);
    free(tmp);
    add(1, 2);
}
