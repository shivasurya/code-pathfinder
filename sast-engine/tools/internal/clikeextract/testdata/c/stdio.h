/* Synthetic stdio.h fixture for clikeextract tests.
 * Mirrors the canonical glibc surface area without the macro magic.
 */
#ifndef _STDIO_H
#define _STDIO_H

typedef struct __FILE FILE;
typedef long fpos_t;

#define EOF (-1)
#define BUFSIZ 8192
#define FILENAME_MAX 4096
#define DEFAULT_PATH "/tmp"

extern FILE* stdin;
extern FILE* stdout;
extern FILE* stderr;

int printf(const char* format, ...);
int fprintf(FILE* stream, const char* format, ...);
int sprintf(char* str, const char* format, ...);
int fclose(FILE* stream);
FILE* fopen(const char* pathname, const char* mode);
size_t fread(void* ptr, size_t size, size_t nmemb, FILE* stream);
size_t fwrite(const void* ptr, size_t size, size_t nmemb, FILE* stream);
int fseek(FILE* stream, long offset, int whence);
long ftell(FILE* stream);
void rewind(FILE* stream);

/* Private symbols — should be filtered out by IsPrivateSymbol. */
int _internal_buffer_flush(FILE*);
int __builtin_printf_check(const char*, ...);

#endif
