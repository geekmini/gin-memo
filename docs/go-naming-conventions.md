# Go Naming Conventions

## Quick Reference

| Type                | Convention              | Example                          |
| ------------------- | ----------------------- | -------------------------------- |
| Package names       | lowercase, single word  | `errors`, `http`, `json`         |
| Import aliases      | lowercase               | `apperrors`, `testify`           |
| Variables           | camelCase               | `userName`, `isValid`            |
| Constants           | camelCase or PascalCase | `maxRetries`, `MaxRetries`       |
| Functions (private) | camelCase               | `parseInput()`, `validateUser()` |
| Functions (public)  | PascalCase              | `ParseInput()`, `ValidateUser()` |
| Structs             | PascalCase              | `User`, `HttpClient`             |
| Interfaces          | PascalCase              | `Reader`, `UserRepository`       |
| Acronyms            | ALL CAPS                | `userID`, `httpURL`, `APIKey`    |

---

## 1. Package Names

**Rule:** Lowercase, single word, no underscores or mixedCaps.

```go
// Good
package errors
package http
package userservice

// Bad
package userService    // No mixedCaps
package user_service   // No underscores
package UserService    // No PascalCase
```

**Tips:**
- Short and concise
- Noun, not verb
- No plural (use `user` not `users`)
- Avoid generic names like `util`, `common`, `misc`

---

## 2. Import Aliases

**Rule:** Lowercase, no mixedCaps.

```go
// Good
import apperrors "gin-sample/internal/errors"
import userrepo "gin-sample/internal/repository"

// Bad
import appErrors "gin-sample/internal/errors"   // No mixedCaps
import user_repo "gin-sample/internal/repository" // No underscores
```

---

## 3. Exported vs Unexported (Public vs Private)

**Rule:** First letter determines visibility.

```go
// Exported (public) - accessible from other packages
type User struct {}           // PascalCase
func NewUser() *User {}       // PascalCase
const MaxRetries = 3          // PascalCase

// Unexported (private) - only accessible within package
type userCache struct {}      // camelCase
func validateEmail() bool {}  // camelCase
const defaultTimeout = 30     // camelCase
```

---

## 4. Variables

**Rule:** camelCase for local variables, PascalCase for exported package variables.

```go
// Local variables - camelCase
userName := "john"
isValid := true
httpClient := &http.Client{}

// Package-level exported - PascalCase
var DefaultTimeout = 30 * time.Second

// Package-level unexported - camelCase
var defaultRetries = 3
```

---

## 5. Acronyms and Initialisms

**Rule:** Keep acronyms consistently cased (usually ALL CAPS).

```go
// Good
userID        // ID is all caps
httpURL       // URL is all caps
apiKey        // API is all caps (but lowercase because unexported)
XMLParser     // XML is all caps
HTMLTemplate  // HTML is all caps

// Bad
UserId        // Inconsistent - should be UserID
HttpUrl       // Inconsistent - should be HTTPURL or httpURL
ApiKey        // Inconsistent - should be APIKey
```

**Common acronyms:**
- `ID` (not `Id`)
- `URL` (not `Url`)
- `HTTP` (not `Http`)
- `API` (not `Api`)
- `JSON` (not `Json`)
- `XML` (not `Xml`)
- `SQL` (not `Sql`)
- `HTML` (not `Html`)

---

## 6. Interfaces

**Rule:** Often end with `-er` suffix for single-method interfaces.

```go
// Single method - use -er suffix
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Stringer interface {
    String() string
}

// Multiple methods - descriptive name
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
}

type HTTPClient interface {
    Get(url string) (*Response, error)
    Post(url string, body []byte) (*Response, error)
}
```

---

## 7. Getters and Setters

**Rule:** Don't use `Get` prefix for getters.

```go
// Good
func (u *User) Name() string { return u.name }
func (u *User) SetName(name string) { u.name = name }

// Bad
func (u *User) GetName() string { return u.name }  // Don't use Get prefix
```

---

## 8. Error Variables

**Rule:** Prefix with `Err`.

```go
// Good
var ErrNotFound = errors.New("not found")
var ErrInvalidInput = errors.New("invalid input")
var ErrUnauthorized = errors.New("unauthorized")

// Bad
var NotFoundError = errors.New("not found")    // Use Err prefix
var InvalidInputErr = errors.New("invalid input") // Err should be prefix, not suffix
```

---

## 9. File Names

**Rule:** Lowercase, use underscores to separate words.

```
// Good
user_repository.go
http_client.go
string_utils.go

// Bad
userRepository.go    // No camelCase
UserRepository.go    // No PascalCase
user-repository.go   // No hyphens
```

**Special suffixes:**
- `_test.go` - Test files
- `_linux.go`, `_windows.go` - Platform-specific
- `_amd64.go`, `_arm64.go` - Architecture-specific

---

## 10. Directory/Package Structure

**Rule:** Lowercase, short names.

```
// Good
/internal/user/
/internal/auth/
/pkg/httputil/

// Bad
/internal/UserService/     // No PascalCase
/internal/user_service/    // Avoid underscores in dirs
```

---

## Summary Cheatsheet

```go
package mypackage                           // lowercase

import apperrors "path/to/errors"           // lowercase alias

var ErrNotFound = errors.New("not found")   // Err prefix

type UserRepository interface {             // PascalCase (exported)
    FindByID(id string) (*User, error)      // ID not Id
}

type userCache struct {                     // camelCase (unexported)
    maxSize int
}

func NewUserRepository() UserRepository {}  // PascalCase (exported)
func validateInput() bool {}                // camelCase (unexported)

func (u *User) Name() string {}             // No Get prefix
func (u *User) SetName(name string) {}      // Set prefix is OK
```

---

## References

- [Effective Go - Names](https://go.dev/doc/effective_go#names)
- [Go Blog - Package Names](https://go.dev/blog/package-names)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
