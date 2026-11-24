// Package op holds the result type of every SDK call, op.Result, and helpers
// for get-or-create, upsert and desired-state operations.
//
// # Methods and functions
//
// Read a result with a method, build one with a function:
//
//   - Everything a caller asks of a result is a method: r.IsError(),
//     r.IsSuccess(), r.IgnoreNotFound(), r.Items(), r.First(), r.Execute(ctx),
//     and so on.
//   - Everything that builds or prepares a result is a package function:
//     op.WithMeta(r, k, v), op.WithHTTPStatus(r, s), op.ExecuteOrDefer(ctx, r),
//     and so on. Mostly the SDK services call these, not their users.
//
// The split is not only a matter of style. Go compiles every method of a
// generic type for each of its instantiations, in every package that refers
// to them. The SDK has over a hundred Result instantiations reachable from
// api.Client, so each method on Result adds over a hundred functions to every
// package that imports the SDK, whether it calls the method or not. A package
// function is only compiled where it is called. The test in
// pkg/c8y/internal/compilecost keeps the count within a budget.
//
// Generic methods (r.Map, r.Decode[U] and so on) are not part of the method
// set, so they cost nothing until they are called.
package op
