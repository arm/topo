# GitHub Copilot Instructions

## Code Style & Structure
- Generate function and variable names that are descriptive and self-documenting
- Avoid generating comments unless explaining *why* something is done, not *what* is being done
- Prefer suggesting pure functions over stateful operations when possible
- Keep generated functions small and focused on a single responsibility
- Use early returns to reduce nesting and improve readability

## Testing Philosophy
- Follow Arrange-Act-Assert structure for test organization
- Generate tests that describe behavior, not implementation
- Include tests for boundaries and edge cases, not just the happy path
- Prefer many small, focused tests over fewer large tests
- Make test names descriptive of the scenario being tested

## Code Quality
- Favor composition over inheritance in suggestions
- Minimize dependencies between modules
- Handle errors explicitly rather than ignoring them
- Use meaningful variable names that explain intent
- Keep cyclomatic complexity low in generated code

## Performance Considerations
- Suggest readable solutions over premature optimization
- Consider algorithmic complexity for data processing
- Recommend caching expensive operations when appropriate

## General Principles
- Generated code should read like well-written prose
- Prioritize clarity and maintainability
- Suggest consistent patterns within the existing codebase
- Delete unused code aggressively

---

## Language-Specific Guidelines

### Golang

#### Testing Best Practices

**Keep Act and Assert Phases Separate**

Don't mix error checking with the action being tested:

```go
// Incorrect - mixing Act and Assert
func TestUserService(t *testing.T) {
    svc := NewUserService()
    got, err := svc.GetUser(1)
    require.NoError(t, err) // This belongs in Assert, not here
}

// Correct - clear separation
func TestUserService(t *testing.T) {
    svc := NewUserService()

    got, err := svc.GetUser(1)

    require.NoError(t, err)
    assert.Equal(t, 1, got.ID)
}
```

**Use Testify Assertions**

Generate tests using [stretchr/testify](https://github.com/stretchr/testify) assertions for clearer test failures.

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestUserCreation(t *testing.T) {
    got := NewUser("Alice", 30)

    assert.Equal(t, "Alice", got.Name)
    assert.Equal(t, 30, got.Age)
    assert.NotNil(t, got.CreatedAt)
}
```

**Prefer Struct Comparison Over Field-by-Field Checks**

Generate tests that compare entire structs in a single assertion rather than checking each field individually.

```go
func TestCreateUser(t *testing.T) {
    got := CreateUser("Bob", "bob@example.com")

    want := User{
        Name:  "Bob",
        Email: "bob@example.com",
        Role:  "user",
    }
    assert.Equal(t, want, got)
}
```

**Use `t.Run()` for Method Tests**

Generate test suites using `t.Run()` with descriptive names. Avoid underscores in test function names.

```go
func TestUserService(t *testing.T) {
    t.Run("GetUser returns user when exists", func(t *testing.T) {
        svc := NewUserService()

        got, err := svc.GetUser(1)

        assert.NoError(t, err)
        assert.Equal(t, 1, got.ID)
    })

    t.Run("GetUser returns error when not found", func(t *testing.T) {
        svc := NewUserService()

        got, err := svc.GetUser(999)

        assert.Error(t, err)
        assert.Nil(t, got)
    })
}
```
