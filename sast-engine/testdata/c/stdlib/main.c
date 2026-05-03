/*
 * Smoke fixture for C stdlib resolution (PR-02).
 *
 * Each call below should resolve against the C stdlib registry:
 *   - printf, fprintf  → stdio.h
 *   - malloc, free     → stdlib.h
 *   - strlen           → string.h
 *
 * The local helper (greet) verifies that user code coexists with
 * stdlib calls without breaking Phase 1 resolution.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void greet(const char *name) {
    printf("hello, %s\n", name);
}

int main(int argc, char **argv) {
    (void)argc;
    (void)argv;

    char *buf = (char *)malloc(64);
    if (buf == NULL) {
        fprintf(stderr, "malloc failed\n");
        return 1;
    }

    const char *who = "world";
    size_t n = strlen(who);
    snprintf(buf, 64, "%s (%zu bytes)", who, n);

    greet(buf);

    free(buf);
    return 0;
}
