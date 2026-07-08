# bizerror

`bizerror` is a small Go package for typed business errors.

It keeps the common fields that service responses usually need:

- numeric business code
- human-readable message
- stable/unstable marker
- wrapped cause
- optional error chain
- optional stack trace
- extra key-value data

The package has no runtime dependencies and does not depend on any RPC framework.

## Install

```sh
go get github.com/wgdl666/kangaroo/bizerr
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/wgdl666/kangaroo/bizerr"
)

var InternalError = bizerror.NewBuilder[int32]().
	SetCode(50001).
	SetMessage("internal error").
	SetStable(false).
	Build()

func main() {
	err := InternalError.
		WithMessageAndStack("create order failed").
		Errorf("db timeout")

	code, ok := bizerror.ParseCode(err)
	fmt.Println(code, ok)
	fmt.Println(bizerror.ParseMessage(err))
}
```

## API

- `NewBuilder[T]()` builds reusable business error definitions from integer-like code types.
- `WithMessage(message)` returns a cloned error with a new message.
- `WithMessageAndStack(message)` returns a cloned error with a new message and current stack.
- `Errorf(format, args...)` appends detail to the error chain and records the first cause.
- `ChainWith(err, format, args...)` appends context to a `BizError`, or wraps ordinary errors.
- `ParseCode(err)` extracts a business code when the error contains a `BizError`.
- `ParseMessage(err)` extracts the business message when possible.
