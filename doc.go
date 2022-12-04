// Decouple analyzes Go packages to find overspecified function parameters.
// If your function takes a *os.File for example,
// but only ever calls Read on it,
// the function can be rewritten to take an io.Reader,
// generalizing the function,
// making it easier to test,
// and decoupling it from whatever the source of the *os.File is.
package decouple
