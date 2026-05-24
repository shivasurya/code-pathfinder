/* Synthetic header exercising inline function definitions, extern "C" blocks,
 * preproc_ifdef/preproc_else branches, struct field declarators, and
 * pointer-returning typedefs. Used to exercise the full set of walkCNode
 * dispatch branches in tests.
 */
#ifndef _INLINE_H
#define _INLINE_H

/* Inline function definition (function_definition node, not declaration). */
static inline int abs_diff(int a, int b) {
    return a > b ? a - b : b - a;
}

/* Function returning a pointer-to-pointer. */
char** make_argv(int argc);

#ifdef __cplusplus
extern "C" {
int legacy_c_only(int x);
}
#endif

/* preproc_else branch: only one branch is taken at compile time, but the
 * AST still includes both. Walker should descend into both arms. */
#ifdef HAVE_FOO
int foo_a(void);
#else
int foo_b(void);
#endif

/* Pointer typedef. */
typedef int* int_ptr;

#endif
