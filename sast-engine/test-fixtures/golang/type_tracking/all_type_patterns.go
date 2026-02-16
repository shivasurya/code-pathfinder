package typetracking

// all_type_patterns.go - Comprehensive test fixture for Go type tracking
// This file covers all patterns that Phase 2 should handle.

// ===== BUILTIN RETURN TYPES =====

// Numeric types.
func GetInt() int {
	return 42
}

func GetInt64() int64 {
	return 123456789
}

func GetFloat64() float64 {
	return 3.14
}

// String and character types.
func GetString() string {
	return "hello"
}

func GetByte() byte {
	return 'A'
}

func GetRune() rune {
	return '日'
}

// Boolean.
func GetBool() bool {
	return true
}

// Error interface.
func GetError() error {
	return nil
}

// ===== POINTER RETURN TYPES =====

func GetUserPointer() *User {
	return &User{ID: 1, Name: "Alice"}
}

func GetConfigPointer() *Config {
	return &Config{Port: 8080}
}

// Double pointer (edge case).
func GetDoublePointer() **User {
	u := &User{ID: 1}
	return &u
}

// ===== MULTI-RETURN TYPES =====

func LoadConfig() (string, error) {
	return "config", nil
}

func ParseInt(s string) (int, bool) {
	return 0, false
}

func GetUserWithError() (*User, error) {
	return &User{ID: 1}, nil
}

// Three returns (extract first).
func ThreeReturns() (string, int, error) {
	return "data", 42, nil
}

// ===== SAME-PACKAGE TYPES =====

func NewUser() *User {
	return &User{}
}

func CreateConfig() Config {
	return Config{Port: 8080}
}

// ===== STRUCT DEFINITIONS =====

type User struct {
	ID   int
	Name string
}

type Config struct {
	Port int
}

// ===== NO RETURN TYPE (void functions) =====

func DoSomething() {
	// No return
}

func PrintMessage(msg string) {
	// No return
}

// ===== METHOD RETURN TYPES =====

func (u *User) GetID() int {
	return u.ID
}

func (u *User) GetName() string {
	return u.Name
}

func (u *User) Clone() *User {
	return &User{ID: u.ID, Name: u.Name}
}

func (u *User) Save() error {
	// Save logic
	return nil
}

func (c *Config) GetPort() int {
	return c.Port
}

func (c *Config) Validate() (bool, error) {
	return true, nil
}
