# Go Concepts Q&A

## 1. What is Context and what is it used for?

### What is it?
`context.Context` is Go's way to carry **deadlines**, **cancellation signals**, and **request-scoped values** across API boundaries and goroutines.

### Why do we need it?
Imagine you make a database call that hangs forever. Without context, your app would wait indefinitely. Context lets you say "give up after 10 seconds."

### Common use cases:
1. **Timeouts** - Cancel operations that take too long
2. **Cancellation** - Stop work when the user cancels a request
3. **Request tracing** - Pass request IDs through the call chain

### Code examples:

```go
// 1. Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()  // Always call cancel!

// This will fail if it takes more than 10 seconds
client.Connect(ctx)
```

```go
// 2. Context with cancellation
ctx, cancel := context.WithCancel(context.Background())

go func() {
    // Do some work...
    if userClickedCancel {
        cancel()  // Signal all operations to stop
    }
}()

// This will stop if cancel() is called
doLongOperation(ctx)
```

```go
// 3. Checking if context was cancelled
select {
case <-ctx.Done():
    return ctx.Err()  // Returns "context deadline exceeded" or "context canceled"
default:
    // Continue working
}
```

### The context hierarchy:

```
context.Background()          <-- Root context (never cancels)
    └── context.WithTimeout() <-- Child: cancels after duration
        └── context.WithValue() <-- Child: carries a value
```

When a parent context is cancelled, all children are cancelled too.

### Rules of thumb:
- `context.Background()` - Use at the top level (main, init, tests)
- `context.TODO()` - Placeholder when you're not sure which context to use
- Always pass context as the **first parameter**: `func DoSomething(ctx context.Context, ...)`
- **Never** store context in a struct; pass it explicitly

---

## 2. What does `defer` mean in Go?

### What is it?
`defer` schedules a function call to run **when the surrounding function returns**. Think of it like `finally` in other languages, but more flexible.

### Why do we need it?
To ensure cleanup code always runs, even if there's an error or early return.

### Basic example:

```go
func readFile() {
    file, err := os.Open("data.txt")
    if err != nil {
        return
    }
    defer file.Close()  // Will run when readFile() exits

    // Read file...
    // Even if we return early or panic, file.Close() will be called
}
```

### Key behaviors:

**1. LIFO order (Last In, First Out)**
```go
func main() {
    defer fmt.Println("1")
    defer fmt.Println("2")
    defer fmt.Println("3")
}
// Output:
// 3
// 2
// 1
```

**2. Arguments are evaluated immediately**
```go
func main() {
    x := 10
    defer fmt.Println(x)  // x is captured as 10 NOW
    x = 20
}
// Output: 10 (not 20!)
```

**3. Can modify named return values**
```go
func double(x int) (result int) {
    defer func() {
        result *= 2  // Modifies the return value
    }()
    return x
}

fmt.Println(double(5))  // Output: 10
```

### Common use cases:

```go
// 1. Close resources
defer file.Close()
defer db.Disconnect()
defer mutex.Unlock()

// 2. Recover from panics
defer func() {
    if r := recover(); r != nil {
        log.Printf("Recovered from panic: %v", r)
    }
}()

// 3. Timing functions
defer func(start time.Time) {
    log.Printf("Took %v", time.Since(start))
}(time.Now())

// 4. Cancel context
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

### defer vs finally (JavaScript):

| JavaScript                          | Go                      |
| ----------------------------------- | ----------------------- |
| `try { ... } finally { cleanup() }` | `defer cleanup()`       |
| One finally block per try           | Multiple defers allowed |
| At the end of try block             | Anywhere in function    |

---

## 3. What is a Receiver Method? What is it used for?

### What is it?
A receiver method is a function attached to a type (struct). It's Go's version of class methods in OOP languages.

### Syntax:

```go
//   receiver    method name    parameters    return type
//      ↓            ↓              ↓             ↓
func (m *MongoDB) Collection(name string) *mongo.Collection {
    return m.Database.Collection(name)
}
```

The `(m *MongoDB)` part is the **receiver**. It attaches this function to the `MongoDB` type.

### Calling receiver methods:

```go
db := &MongoDB{...}
users := db.Collection("users")  // Call method on db instance
```

### Two types of receivers:

**1. Pointer receiver `(m *MongoDB)`**
```go
func (m *MongoDB) Close() {
    m.Client = nil  // CAN modify the struct
}
```
- Can modify the struct
- Avoids copying large structs
- Use this most of the time

**2. Value receiver `(m MongoDB)`**
```go
func (m MongoDB) GetName() string {
    return m.Name  // Cannot modify m
}
```
- Gets a copy of the struct
- Cannot modify the original
- Use for small, immutable structs

### Why use receiver methods?

**1. Organize code around data**
```go
type User struct {
    Name  string
    Email string
}

// Methods belong to User
func (u *User) Validate() error { ... }
func (u *User) Save() error { ... }
func (u *User) SendEmail() error { ... }
```

**2. Implement interfaces**
```go
type Stringer interface {
    String() string
}

func (u *User) String() string {
    return fmt.Sprintf("User: %s", u.Name)
}

// Now User implements Stringer interface
```

**3. Encapsulation**
```go
type Counter struct {
    count int  // lowercase = private
}

func (c *Counter) Increment() {
    c.count++  // Only accessible via method
}

func (c *Counter) Value() int {
    return c.count
}
```

### Comparison with other languages:

| Language   | Syntax                     |
| ---------- | -------------------------- |
| Go         | `func (u *User) Save() {}` |
| JavaScript | `class User { save() {} }` |
| Python     | `def save(self): ...`      |
| Java       | `public void save() {}`    |

### When to use pointer vs value receiver:

| Use Pointer `*T`                          | Use Value `T`                |
| ----------------------------------------- | ---------------------------- |
| Method modifies the receiver              | Method only reads data       |
| Struct is large                           | Struct is small (few fields) |
| Consistency (if one method needs pointer) | Immutable types              |
| 90% of the time                           | Rare cases                   |

**Rule of thumb:** When in doubt, use a pointer receiver.
