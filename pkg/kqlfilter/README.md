# KQL filter package

This package contains Kibana Query Language parser.

```bash
go get "github.com/mycujoo/go-stdlib/pkg/kqlfilter"
```

## Usage

You can use either `Parse` or `ParseAST` to parse a KQL filter.

`Parse` will return a `Filter` struct, which is simple to use, but does not support all KQL features.
```go
package main

import (
    "fmt"

    "github.com/mycujoo/go-stdlib/pkg/kqlfilter"
)

func main() {
    filter, err := kqlfilter.Parse("foo:bar", false)
    if err != nil {
        panic(err)
    }

    fmt.Println(filter)
}
```

`ParseAST` will return an `AST` struct, which is more complex to use, but supports all KQL features.
It returns an `AST` struct, which is a tree of `Node`s.
```go
package main

import (
    "fmt"

    "github.com/mycujoo/go-stdlib/pkg/kqlfilter"
)

func main() {
    ast, err := kqlfilter.ParseAST("foo:bar")
    if err != nil {
        panic(err)
    }

    fmt.Println(ast)
}
```
