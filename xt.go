/*
XML Types. Missing feature of the `encoding/xml` package: generic representation
of arbitrary XML documents and nodes, mostly-reversible, with JSON support.

• Decodes arbitrary XML with minimal information loss.

• Encodes back into XML, not identical but equivalent to original.

• Encodes and decodes as JSON with no information loss.

See `readme.md` for examples.
*/
package xt

// Spec: https://www.w3.org/TR/2008/REC-xml-20081126/

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"unsafe"
)

// Types of XML nodes, used in JSON.
const (
	TypePi      = "pi"
	TypeDecl    = "decl"
	TypeComment = "comment"
	TypeText    = "text"
	TypeElem    = "elem"
)

/*
Represents an XML name with the namespace and local parts. Variant of `xml.Name`
with JSON support.

XML <-> JSON (for full element):

	<one:two xmlns:one="three" />
	<->
	{"type": "elem", "name": {"space": "three", "local": "two"}}
	<->
	<two xmlns="three" />
*/
type Name struct {
	Space string `json:"space,omitempty"`
	Local string `json:"local,omitempty"`
}

/*
Represents an XML attribute. Variant of `xml.Attr` with JSON support.

XML <-> JSON:

	one="two"
	<->
	{"name": {"local": "one"}, "value": "two"}
*/
type Attr struct {
	Name  Name   `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

/*
Represents any XML node. One of:

	* Pi
	* Decl
	* Comment
	* Text
	* Elem
	* Nodes

Obtained via `Decode`, `DecodeToken`, `(*Nodes).Decode`, `(*Nodes).DecodeToken`.
*/
type Node interface {
	xml.Marshaler
	json.Marshaler
}

/*
Short for "processing instruction". Variant of `xml.ProcInst` with reversible
decoding/encoding and JSON support.

XML <-> JSON:

	<?xml one two three?>
	<->
	{"type": "pi", "target": "xml", "content": "one two three"}

To preserve such nodes, use `(*Nodes).Decode` rather than `xml.Unmarshal`.
*/
type Pi struct {
	Target  string `json:"target,omitempty"`
	Content string `json:"content,omitempty"`
}

var _ = xml.Marshaler(Pi{})

func (self Pi) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	if self.Target == "" {
		return fmt.Errorf(`can't encode XML processing instruction with empty target`)
	}

	return enc.EncodeToken(xml.ProcInst{
		Target: self.Target,
		// TODO: consider avoiding string-to-bytes allocation. Benchmark first.
		Inst: []byte(self.Content),
	})
}

func (self Pi) MarshalJSON() ([]byte, error) {
	type inner Pi
	return json.Marshal(struct {
		typeHead
		inner
	}{typeHead{TypePi}, inner(self)})
}

/*
Represents an arbitrary XML declaration, such as "<!DOCTYPE>". Variant of
`xml.Directive` with reversible decoding/encoding and JSON support. Renamed
to "declaration" to match the lexicon of the XML spec.

XML <-> JSON:

	<!one two three>
	<->
	{"type": "decl", "content": "one two three"}

To preserve such nodes, use `(*Nodes).Decode` rather than `xml.Unmarshal`.
*/
type Decl string

var _ = xml.Marshaler(Decl(""))

func (self Decl) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	// TODO: consider avoiding string-to-bytes allocation. Benchmark first.
	return enc.EncodeToken(xml.Directive(self))
}

func (self *Decl) UnmarshalJSON(input []byte) error {
	return jsonUnmarshalContent(input, (*string)(self))
}

func (self Decl) MarshalJSON() ([]byte, error) {
	return jsonMarshalContent(TypeDecl, string(self))
}

/*
Represents an XML comment. Variant of `xml.Comment` with reversible
decoding/encoding.

XML <-> JSON:

	<!-- some text -->
	<->
	{"type": "comment", "content": " some text "}

To preserve such nodes, use `(*Nodes).Decode` rather than `xml.Unmarshal`.
*/
type Comment string

var _ = xml.Marshaler(Comment(""))

func (self Comment) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	// TODO: consider avoiding string-to-bytes allocation. Benchmark first.
	return enc.EncodeToken(xml.Comment(self))
}

func (self *Comment) UnmarshalJSON(input []byte) error {
	return jsonUnmarshalContent(input, (*string)(self))
}

func (self Comment) MarshalJSON() ([]byte, error) {
	return jsonMarshalContent(TypeComment, string(self))
}

/*
Represents XML text. Variant of `xml.CharData` with reversible decoding/encoding
and JSON support.

XML <-> JSON:

	some content
	<->
	{"type": "text", "content": "some content"}
*/
type Text string

var _ = xml.Marshaler(Text(""))

func (self Text) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	// TODO: consider avoiding string-to-bytes allocation. Benchmark first.
	return enc.EncodeToken(xml.CharData(self))
}

func (self *Text) UnmarshalJSON(input []byte) error {
	return jsonUnmarshalContent(input, (*string)(self))
}

func (self Text) MarshalJSON() ([]byte, error) {
	return jsonMarshalContent(TypeText, string(self))
}

/*
Represents an arbitrary XML element with minimal information loss.
*/
type Elem struct {
	Name  Name   `json:"name,omitempty"`
	Attrs []Attr `json:"attrs,omitempty"`
	Nodes Nodes  `json:"nodes,omitempty"`
}

var _ = xml.Unmarshaler((*Elem)(nil))

func (self *Elem) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	self.Name = Name(start.Name)
	self.Attrs = attrsFrom(start.Attr)

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		_, ok := tok.(xml.EndElement)
		if ok {
			return nil
		}

		err = self.Nodes.DecodeToken(dec, tok)
		if err != nil {
			return err
		}
	}
}

var _ = xml.Marshaler(Elem{})

func (self Elem) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	/**
	This prevents the XML package from encoding an `Elem` with an empty name as
	<Elem />. Because this is a generic representation of an arbitrary XML
	element, its own type should never leak into the resulting document.

	We could encode as `""` when `self == Elem{}`, but there are malformed
	states we can't support, such as having an empty name but non-empty
	attributes and child nodes. It's simpler to just forbid empty names.
	*/
	if self.Name.Local == "" {
		return fmt.Errorf(`can't XML-encode %T with empty name`, self)
	}

	/**
	Workaround for the unfortunate propensity of `encoding/xml` to produce
	redundant `xmlns` attributes, which would accumulate over the course of
	repeated decoding and encoding.
	*/
	if hasExactAttr(self.Attrs, "", "xmlns", self.Name.Space) {
		self.Name.Space = ""
	}

	start := xml.StartElement{Name: xml.Name(self.Name), Attr: attrsTo(self.Attrs)}
	err := enc.EncodeToken(start)
	if err != nil {
		return err
	}

	err = self.Nodes.MarshalXML(enc, xml.StartElement{})
	if err != nil {
		return err
	}

	return enc.EncodeToken(start.End())
}

func (self Elem) MarshalJSON() ([]byte, error) {
	type inner Elem
	return json.Marshal(struct {
		typeHead
		inner
	}{typeHead{TypeElem}, inner(self)})
}

/*
Decodes an arbitrary XML node, starting at the given token. If the token is
`xml.StartElement`, this consumes and parses the entire element, including its
child nodes and `xml.EndElement`.

Also see `(*Nodes).Decode` and `(*Nodes).DecodeToken`.
*/
func DecodeToken(dec *xml.Decoder, tok xml.Token, out *Node) error {
	switch tok := tok.(type) {
	case xml.ProcInst:
		*out = Pi{tok.Target, string(tok.Inst)}
		return nil

	case xml.Directive:
		*out = Decl(tok)
		return nil

	case xml.Comment:
		*out = Comment(tok)
		return nil

	case xml.CharData:
		*out = Text(tok)
		return nil

	case xml.StartElement:
		var elem Elem
		err := dec.DecodeElement(&elem, &tok)
		if err != nil {
			return err
		}
		*out = elem
		return nil
	}

	return fmt.Errorf(`unexpected XML token %#v`, tok)
}

/*
Represents a sequence of arbitrary XML nodes. In XML, decodes and encodes with
no wrappers or separators. In JSON, decodes and encodes as an array.

`Nodes` is also the representation of an arbitrary top-level XML document. It
preserves processing instructions such as `<?xml?>`, declarations such as
`<!DOCTYPE>`, and so on.

XML <-> JSON:

	<?xml one?>
	<two />
	<three />

	<->

	[
		{"type": "pi", "target": "xml", "content": "one"},
		{"type": "elem", "name": {"local": "two"}, "attrs": null, "nodes": null},
		{"type": "elem", "name": {"local": "three"}, "attrs": null, "nodes": null}
	]
*/
type Nodes []Node

/*
Decodes an arbitrary sequence of XML nodes, which may be a top-level XML
document. To encode it back, simply pass the resulting `Nodes` to `xml.Marshal`
or `(*xml.Encoder).Encode`.
*/
func (self *Nodes) Decode(dec *xml.Decoder) error {
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		var node Node
		err = DecodeToken(dec, tok, &node)
		if err != nil {
			return err
		}
		*self = append(*self, node)
	}
}

/*
Decodes an arbitrary XML node via `DecodeToken` and appends it to the sequence.
*/
func (self *Nodes) DecodeToken(dec *xml.Decoder, tok xml.Token) error {
	var node Node
	err := DecodeToken(dec, tok, &node)
	if err != nil {
		return err
	}
	*self = append(*self, node)
	return nil
}

var _ = xml.Unmarshaler((*Nodes)(nil))

func (self *Nodes) UnmarshalXML(dec *xml.Decoder, _ xml.StartElement) error {
	return self.Decode(dec)
}

var _ = xml.Marshaler(Nodes(nil))

func (self Nodes) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	for _, node := range self {
		err := enc.Encode(node)
		if err != nil {
			return err
		}
	}
	return nil
}

var _ = json.Unmarshaler((*Nodes)(nil))

func (self *Nodes) UnmarshalJSON(input []byte) error {
	return json.Unmarshal(input, (*[]nodeDecoder)(unsafe.Pointer(self)))
}

func jsonUnmarshalContent(input []byte, out *string) error {
	return json.Unmarshal(input, &struct {
		Content *string `json:"content,omitempty"`
	}{out})
}

func jsonMarshalContent(typ string, content string) ([]byte, error) {
	return json.Marshal(struct {
		typeHead
		Content string `json:"content,omitempty"`
	}{typeHead{typ}, content})
}

func attrsFrom(attrs []xml.Attr) []Attr {
	return *(*[]Attr)(unsafe.Pointer(&attrs))
}

func attrsTo(attrs []Attr) []xml.Attr {
	return *(*[]xml.Attr)(unsafe.Pointer(&attrs))
}

type typeHead struct {
	Type string `json:"type,omitempty"`
}

type nodeDecoder struct{ Node }

func (self *nodeDecoder) UnmarshalJSON(input []byte) error {
	var head typeHead
	err := json.Unmarshal(input, &head)
	if err != nil {
		return err
	}

	if head.Type == "" {
		return fmt.Errorf(`required field "type" is missing in %q`, input)
	}

	switch head.Type {
	case TypePi:
		var val Pi
		err = json.Unmarshal(input, &val)
		self.Node = val

	case TypeDecl:
		var val Decl
		err = json.Unmarshal(input, &val)
		self.Node = val

	case TypeComment:
		var val Comment
		err = json.Unmarshal(input, &val)
		self.Node = val

	case TypeText:
		var val Text
		err = json.Unmarshal(input, &val)
		self.Node = val

	case TypeElem:
		var val Elem
		err = json.Unmarshal(input, &val)
		self.Node = val

	default:
		err = fmt.Errorf(`unrecognized node type %q`, head.Type)
	}
	return err
}

func hasExactAttr(attrs []Attr, space string, local string, value string) bool {
	for _, attr := range attrs {
		if attr.Name.Space == space && attr.Name.Local == local && attr.Value == value {
			return true
		}
	}
	return false
}
