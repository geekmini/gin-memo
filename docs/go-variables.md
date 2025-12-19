# Variables in Go: `:=` vs `=`

## Quick Reference

| Operator | Name | Purpose |
|----------|------|---------|
| `:=` | Short declaration | Declare AND assign (new variable) |
| `=` | Assignment | Assign to existing variable |

---

## `:=` Short Declaration

Creates a **new variable** and assigns a value. Go infers the type.

```go
name := "John"           // Creates new variable, type inferred as string
age := 25                // Creates new variable, type inferred as int
user := User{Name: "X"}  // Creates new variable, type inferred as User
```

**Equivalent to:**

```go
var name string = "John"
var age int = 25
var user User = User{Name: "X"}
```

**Rules:**
- Only works inside functions (not at package level)
- Must create at least one NEW variable
- Type is inferred from the right side

---

## `=` Assignment

Assigns a value to an **existing variable**.

```go
var name string  // Declare first
name = "John"    // Then assign

age := 25        // Declare with :=
age = 30         // Reassign with =
```

---

## Comparison

```go
// := declares AND assigns (new variable)
x := 10

// = only assigns (variable must exist)
x = 20

// This would error:
y = 30  // ERROR: undefined: y (y doesn't exist yet)
```

---

## Multiple Variables

```go
// Declare multiple with :=
name, age := "John", 25

// Reassign multiple with =
name, age = "Jane", 30

// Mixed: at least one must be NEW for :=
name, email := "John", "john@example.com"  // email is new, name is reassigned
```

---

## Common Patterns

### Function Returns

```go
// Single return
result := doSomething()

// Multiple returns
value, err := getSomething()
if err != nil {
    return err
}
```

### Reassigning with Error

```go
user, err := getUser("123")    // Both new
profile, err := getProfile()   // profile is new, err is reassigned (ok!)
```

This works because `:=` requires **at least one** new variable.

### if Statement Scope

```go
// x only exists inside the if block
if x := getValue(); x > 10 {
    fmt.Println(x)
}
// x doesn't exist here
```

---

## Package Level Variables

At package level, you **must** use `var`:

```go
package main

var globalName = "John"     // OK - var at package level
globalAge := 25             // ERROR - := not allowed at package level

func main() {
    localName := "Jane"     // OK - := inside function
}
```

---

## Type Declaration

```go
// := infers type
name := "John"              // type: string
age := 25                   // type: int
price := 19.99              // type: float64

// var with explicit type
var name string = "John"
var count int64 = 100       // Specific int type

// var without initial value (zero value)
var name string             // "" (empty string)
var age int                 // 0
var active bool             // false
var user *User              // nil
```

---

## Zero Values

When using `var` without assignment, Go assigns "zero values":

```go
var s string    // ""
var i int       // 0
var f float64   // 0.0
var b bool      // false
var p *User     // nil
var sl []int    // nil
var m map[string]int  // nil
```

---

## Shadowing (Caution!)

```go
name := "John"

if true {
    name := "Jane"      // NEW variable in this scope (shadows outer)
    fmt.Println(name)   // "Jane"
}

fmt.Println(name)       // "John" (outer variable unchanged)
```

**To modify outer variable:**

```go
name := "John"

if true {
    name = "Jane"       // = assigns to existing outer variable
    fmt.Println(name)   // "Jane"
}

fmt.Println(name)       // "Jane" (outer variable changed)
```

---

## Comparison with Other Languages

| Go | JavaScript | Python |
|----|------------|--------|
| `x := 10` | `let x = 10` | `x = 10` |
| `x = 20` | `x = 20` | `x = 20` |
| `var x int` | `let x` | N/A |

**Key difference:** Go requires explicit declaration (`:=` or `var`), catches typos at compile time.

```go
userName := "John"
userNmae = "Jane"   // ERROR: undefined: userNmae (typo caught!)
```

```python
user_name = "John"
user_nmae = "Jane"  # No error, creates new variable (bug!)
```

---

## Summary

| Use | When |
|-----|------|
| `:=` | Creating a new variable inside a function |
| `=` | Reassigning an existing variable |
| `var` | Package level, explicit type, or zero value needed |

```go
// Inside functions
name := "John"       // New variable (preferred)
name = "Jane"        // Reassign

// Package level
var AppName = "MyApp"  // Must use var

// Explicit type needed
var count int64 = 100  // Can't infer int64 with :=

// Zero value needed
var user *User         // nil pointer
```
