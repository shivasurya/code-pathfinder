/* Synthetic unistd.h fixture: a few POSIX functions and a #define. */
#ifndef _UNISTD_H
#define _UNISTD_H

typedef long ssize_t;
typedef int pid_t;

#define STDIN_FILENO 0
#define STDOUT_FILENO 1
#define STDERR_FILENO 2

ssize_t read(int fd, void* buf, size_t count);
ssize_t write(int fd, const void* buf, size_t count);
int close(int fd);
int open(const char* pathname, int flags, int mode);
pid_t fork(void);
int execvp(const char* file, char* const argv[]);

#endif
