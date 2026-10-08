# Rules

## Testing

- Write Go tests as **table-driven tests**: a slice of `{name, input, want}` cases run as
  `t.Run` subtests in a loop. One `TestXxx` per unit, cases (happy path, edge cases,
  invalid input) as rows — not separate `TestXxxYyy` functions.
- Set per-case env with `t.Setenv`; assert `wantErr` cases with `t.Fatal` on missing error.
- Reference example: `system/go/common/client/questdb/config_test.go` (`TestNewConfig`).
