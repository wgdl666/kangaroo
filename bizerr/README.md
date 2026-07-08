# bizerr

`bizerr` is a small Go package for business errors with a code, stack, and wrapped cause.

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

func main() {
	cause := fmt.Errorf("db timeout")
	err := bizerr.Wrap(50001, cause, "create order failed")

	code, ok := bizerr.Code(err)
	fmt.Println(code, ok)
	fmt.Println(bizerr.Message(err))
	fmt.Println(bizerr.Stack(err))
}
```

## API

- `New(code, message)` creates an error and records the current stack.
- `Errorf(code, format, args...)` creates a formatted error and records the current stack.
- `Wrap(code, err, message)` wraps a cause with `%w` semantics and records the current stack.
- `Wrapf(code, err, format, args...)` wraps a cause with a formatted message.
- `Code(err)`, `Message(err)`, and `Stack(err)` extract fields from any wrapped `bizerr.Error`.
