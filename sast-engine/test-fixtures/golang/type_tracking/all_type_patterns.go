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

// ===== VARIABLE ASSIGNMENTS =====

// DemoVariableAssignments demonstrates all variable assignment patterns
// that Phase 2 should track for type inference.
func DemoVariableAssignments() {
	// Function call assignments - type from return type
	user := GetUserPointer()        // *User
	config := CreateConfig()        // Config
	name, err := LoadConfig()       // string, error (multi-assignment)
	intVal := GetInt()              // int
	boolVal := GetBool()            // bool

	// Literal assignments - type from literal
	str := "hello"                  // builtin.string
	num := 42                       // builtin.int
	floatNum := 3.14                // builtin.float64
	flag := true                    // builtin.bool
	falsyFlag := false              // builtin.bool

	// Variable reference assignments - type copied from original
	user2 := user                   // *User (copied from user)
	config2 := config               // Config (copied from config)

	// Struct literal assignments - type from literal type
	u := User{ID: 1, Name: "Alice"} // User
	c := Config{Port: 8080}         // Config

	// Pointer to struct literal - type from literal type
	userPtr := &User{ID: 2, Name: "Bob"} // *User
	configPtr := &Config{Port: 9090}     // *Config

	// Reassignment - creates new binding for same variable
	user = &User{ID: 3}             // *User (new binding)
	config = Config{Port: 3000}     // Config (new binding)

	// Suppress unused variable warnings
	_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _ = user2, config2, u, c, userPtr, configPtr, str, num, floatNum, flag, falsyFlag, intVal, boolVal, name, err, user, config
}

// GetTwoInts returns two integers for multi-assignment testing.
func GetTwoInts() (int, int) {
	return 1, 2
}

// DemoComplexAssignments demonstrates more complex assignment patterns.
func DemoComplexAssignments() {
	// Multi-assignment from function
	x, y := GetTwoInts()            // int, int

	// Assignment from method call
	user := GetUserPointer()
	id := user.GetID()              // int
	name := user.GetName()          // string

	// Chained assignments
	a := GetInt()                   // int
	b := a                          // int (copied)
	c := b                          // int (copied again)

	// Suppress unused variable warnings
	_, _, _, _, _, _ = x, y, id, name, a, c
}
