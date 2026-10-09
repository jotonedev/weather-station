# Rules

## Code quality

- Use `time.Time` for instants of time and `time.Duration` for periods, not raw ints. Do all time math with the `"time"` package.

## Errors

- Declare errors so callers can match them when needed. Use a top-level `var ... = errors.New(...)` for static messages that callers match with `errors.Is`, a custom `error` type for dynamic messages they match with `errors.As`, and `fmt.Errorf` otherwise.
- Wrap errors with context using `%w` when the caller should match the cause, `%v` when it should not. Keep context succinct, avoid phrases like `failed to`.
- Prefix exported error vars with `Err` and unexported ones with `err`. Suffix custom error types with `Error`.
- Handle each error once. Do not log an error and then return it, since callers further up may also handle it.

  ```go
  u, err := getUser(id)
  if err != nil {
    return fmt.Errorf("get user %q: %w", id, err)
  }
  ```

## Don't panic

- Return errors instead of panicking. Only panic for irrecoverable conditions.
- In tests prefer `t.Fatal` or `t.FailNow` over panic.

## Type assertions

- Always use the comma-ok form of type assertions. The single-value form panics on failure.

  ```go
  t, ok := i.(string)
  if !ok {
    // handle the failure
  }
  ```

## Defer

- Use `defer` to clean up resources and release locks. It is easy to miss manual unlocks and closes across multiple returns.

  ```go
  p.Lock()
  defer p.Unlock()
  ```

## Avoid mutable globals

- Prefer dependency injection over globals whose values change. This applies to function pointers as well as other values, and makes behavior testable.

## Avoid init()

- Prefer explicit constructors and logic in `main()` over `init()`. `init()` makes ordering and behavior hard to reason about.

## Goroutines

- Do not fire-and-forget goroutines (relevant in data-ingest). Every goroutine needs a managed lifetime, either a predictable stop time or a way to signal it to stop, plus a way to wait for it to exit (e.g. `close(stop)` then `<-done`, or a `sync.WaitGroup`). Consider `go.uber.org/goleak` in tests.

## Marshaled structs

- Annotate struct fields with `json` and `yaml` tags. The serialized form is a contract between systems, so fixed field names guard against breaking it.

## File organization

- Functions that map one value or data structure to another belong in a file named `mapper.go`.

## Built-in names

- Do not shadow predeclared identifiers like `error`, `string`, `len`, `cap`. Shadowing creates confusing, hard-to-grep bugs.

## Reduce nesting

- Handle error and special cases first, returning early or continuing the loop. Avoid `else` after a branch that returns or continues.

  ```go
  for _, v := range data {
    if v.F1 != 1 {
      log.Printf("Invalid v: %v", v)
      continue
    }
    v = process(v)
    if err := v.Call(); err != nil {
      return err
    }
    v.Send()
  }
  ```

## Struct literals

- Specify field names when initializing structs (enforced by `go vet`).

  ```go
  k := User{
    FirstName: "John",
    LastName:  "Doe",
  }
  ```

  Omit fields that are the anonymous zero value. Exception: field names may be omitted in test tables with three or fewer fields.

## Testing

- Write Go tests as table-driven tests, a slice of `{name, input, want}` cases run as `t.Run` subtests in a loop. One `TestXxx` per unit, cases (happy path, edge cases, invalid input) as rows, not separate `TestXxxYyy` functions.
- Set per-case env with `t.Setenv`, assert `wantErr` cases with `t.Fatal` on missing error.
