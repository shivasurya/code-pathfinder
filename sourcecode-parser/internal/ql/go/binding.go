package ql

// #cgo CFLAGS: -std=c11 -fPIC
// #include "../src/parser.c"
// // NOTE: if your language has an external scanner, add it here.
import "C"

import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_ql())
	return sitter.NewLanguage(ptr)
}
