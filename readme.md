## Overview

**X**ML **T**ypes. Missing feature of the `encoding/xml` package: generic representation of arbitrary XML documents and nodes, mostly-reversible, with JSON support.

* Decodes arbitrary XML with minimal information loss.
* Encodes back into XML, not identical but equivalent to original.
* Encodes and decodes as JSON with no information loss.

Small and dependency-free. The dependencies in `go.mod` are test-only.

See API docs at https://pkg.go.dev/github.com/purelabio/xt.

## Usage

This example will decode an XML, re-encode as XML, encode as JSON.

```golang
import (
  "bytes"
  "encoding/json"
  "encoding/xml"
  "os"

  "github.com/purelabio/xt"
)

func main() {
  var nodes xt.Nodes
  err := nodes.Decode(xml.NewDecoder(bytes.NewReader(src)))
  if err != nil {
    panic(err)
  }

  err = xml.NewEncoder(os.Stdout).Encode(nodes)
  if err != nil {
    panic(err)
  }

  err = json.NewEncoder(os.Stdout).Encode(nodes)
  if err != nil {
    panic(err)
  }
}

var src = []byte(`<?xml version="1.0"?>
<one two="three">
  four
</one>
`)
```

Resulting data structure:

```golang
xt.Nodes{
  xt.Pi{Target: "xml", Content: "version=\"1.0\""},
  xt.Text("\n"),
  xt.Elem{
    Name: xt.Name{Local: "one"},
    Attrs: []xt.Attr{
      {Name: xt.Name{Local: "two"}, Value: "three"},
    },
    Nodes: xt.Nodes{
      xt.Text("\n  four\n"),
    },
  },
  xt.Text("\n"),
}
```

## Limitations

* Limitation of `encoding/xml`: doesn't preserve short namespace prefixes. When serializing, it inlines `xmlns` attributes everywhere. The resulting XML should be semantically equivalent to the original, even if the representation is different.

* Limitation of `encoding/xml`: doesn't preserve `<![CDATA[]]>`. All text is serialized as regular text, using escape sequences as appropriate. Again, the result should be semantically equivalent to the original.

* Support for token streaming is limited. `DecodeToken` can decode non-element nodes one-by-one, but always consumes and allocates the entire content of an element, without the ability to "step in" and "step out".

## License

https://unlicense.org
