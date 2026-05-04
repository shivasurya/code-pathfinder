/* Synthetic glibc-style string.h fixture exercising the attribute macro
 * preprocess pipeline. Mirrors the real /usr/include/string.h shape on
 * Debian/Ubuntu for the symbols PR-04 must recover.
 *
 * Keep declarations close to the real glibc spelling — the test exists
 * specifically to catch regressions where the preprocess regex drifts
 * out of sync with what glibc actually emits.
 */
#ifndef _GLIBC_STRING_H
#define _GLIBC_STRING_H

typedef unsigned long size_t;

/*
 * NOTE: real glibc headers don't define these macros locally — they come
 * from <sys/cdefs.h>, which we don't include here on purpose. The
 * preprocess step in clike_preprocess.go strips them from the source
 * before tree-sitter parses, so the token-level result is the same as if
 * cdefs.h had been included and they'd expanded to nothing.
 */

extern size_t strlen (const char *__s)
     __THROW __attribute_pure__ __nonnull ((1));

extern int strcmp (const char *__s1, const char *__s2)
     __THROW __attribute_pure__ __nonnull ((1, 2));

extern int strncmp (const char *__s1, const char *__s2, size_t __n)
     __THROW __attribute_pure__ __nonnull ((1, 2));

extern int strcasecmp (const char *__s1, const char *__s2)
     __THROW __attribute_pure__ __nonnull ((1, 2));

extern void *memcmp (const void *__s1, const void *__s2, size_t __n)
     __THROW __attribute_pure__ __nonnull ((1, 2));

extern void *memmem (const void *__haystack, size_t __haystacklen,
		     const void *__needle, size_t __needlelen)
    __THROW __attribute_pure__ __nonnull ((1, 3))
    __attr_access ((__read_only__, 1, 2))
    __attr_access ((__read_only__, 3, 4));

extern char *strerror_r (int __errnum, char *__buf, size_t __buflen)
     __THROW __nonnull ((2)) __wur;

/* The __THROWNL form (no-throw, no-longjmp) — used by snprintf. */
extern int snprintf (char *__restrict __s, size_t __maxlen,
		     const char *__restrict __format, ...)
     __THROWNL __attribute__ ((__format__ (__printf__, 3, 4)));

extern int vsnprintf (char *__restrict __s, size_t __maxlen,
		      const char *__restrict __format, void *__arg)
     __THROWNL __attribute__ ((__format__ (__printf__, 3, 0)));

#endif
