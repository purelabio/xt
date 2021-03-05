package xt

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleDecode(t *testing.T) {
	src := read(t, `simple.xml`)

	var doc Nodes
	require.NoError(t, doc.Decode(xml.NewDecoder(bytes.NewReader(src))))
	require.Equal(t, expectedSimple, doc)
}

func TestSimpleEncode(t *testing.T) {
	src := read(t, `simple.xml`)

	var doc Nodes
	require.NoError(t, doc.Decode(xml.NewDecoder(bytes.NewReader(src))))
	require.Equal(t, expectedSimple, doc)

	content, err := xml.Marshal(doc)
	require.NoError(t, err)
	require.Equal(t, src, content)
}

func TestSimpleJsonDecode(t *testing.T) {
	src := read(t, `simple.json`)

	var doc Nodes
	require.NoError(t, json.Unmarshal(src, &doc))
	require.Equal(t, expectedSimple, doc)
}

func TestSimpleJsonEncode(t *testing.T) {
	src := read(t, `simple.json`)

	var doc Nodes
	require.NoError(t, json.Unmarshal(src, &doc))
	require.Equal(t, expectedSimple, doc)

	out, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	require.Equal(t, src, out)
}

func TestSimpleXmlToJson(t *testing.T) {
	src := read(t, `simple.xml`)

	var doc Nodes
	require.NoError(t, doc.Decode(xml.NewDecoder(bytes.NewReader(src))))
	require.Equal(t, expectedSimple, doc)

	out, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)

	expected := read(t, `simple.json`)
	require.Equal(t, expected, out)
}

func TestNsDecodeAliased(t *testing.T) {
	src := read(t, `ns_aliased.xml`)

	var doc Nodes
	require.NoError(t, doc.Decode(xml.NewDecoder(bytes.NewReader(src))))
	require.Equal(t, expectedNsAliased, doc)
}

func TestNsDecodeInlined(t *testing.T) {
	src := read(t, `ns_inlined.xml`)

	var doc Nodes
	require.NoError(t, doc.Decode(xml.NewDecoder(bytes.NewReader(src))))
	require.Equal(t, expectedNsInlined, doc)
}

func TestNsEncode(t *testing.T) {
	src := read(t, `ns_inlined.xml`)

	var doc Nodes
	require.NoError(t, doc.Decode(xml.NewDecoder(bytes.NewReader(src))))
	require.Equal(t, expectedNsInlined, doc)

	out := read(t, `ns_out.xml`)

	content, err := xml.Marshal(doc)
	require.NoError(t, err)
	require.Equal(t, out, content)
}

var expectedSimple = Nodes{
	Pi{
		Target:  `xml`,
		Content: `version="1.0" encoding="utf-8"`,
	},
	Text("\n"),
	Elem{
		Name:  Name{Local: "one"},
		Attrs: []Attr{{Name: Name{Local: "two"}, Value: "three"}},
		Nodes: Nodes{
			Text("\n  five\n  "),
			Elem{
				Name:  Name{Local: "six"},
				Attrs: []Attr{{Name: Name{Local: "seven"}, Value: "eight"}},
				Nodes: Nodes{
					Text("\n    "),
					Elem{
						Name:  Name{Local: "nine"},
						Attrs: []Attr{{Name: Name{Local: "ten"}, Value: "eleven"}},
						Nodes: Nodes{
							Text("\n      twelve\n      "),
							Comment(" thirteen "),
							Text("\n    "),
						},
					},
					Text("\n    fourteen\n  "),
				},
			},
			Text("\n  sixteen\n  "),
			Comment(" seventeen "),
			Text("\n"),
		},
	},
}

var expectedNsAliased = Nodes{
	Pi{
		Target:  `xml`,
		Content: `version="1.0" encoding="utf-8"`,
	},
	Text("\n"),
	Elem{
		Name: Name{Space: `ns_outer`, Local: `one`},
		Attrs: []Attr{
			{Name: Name{Space: `xmlns`, Local: `outer`}, Value: `ns_outer`},
			{Name: Name{Local: `two`}, Value: `three`},
		},
		Nodes: Nodes{
			Text("\n  "),
			Elem{
				Name:  Name{Space: `ns_outer`, Local: `four`},
				Attrs: []Attr{},
			},
			Text("\n  "),
			Elem{
				Name: Name{Space: `ns_inner`, Local: `five`},
				Attrs: []Attr{
					{Name: Name{Space: `xmlns`, Local: `inner`}, Value: `ns_inner`},
					{Name: Name{Local: `six`}, Value: `seven`},
				},
			},
			Text("\n"),
		},
	},
}

var expectedNsInlined = Nodes{
	Pi{
		Target:  `xml`,
		Content: `version="1.0" encoding="utf-8"`,
	},
	Text("\n"),
	Elem{
		Name: Name{Space: `ns_outer`, Local: `one`},
		Attrs: []Attr{
			{Name: Name{Local: `xmlns`}, Value: `ns_outer`},
			{Name: Name{Local: `two`}, Value: `three`},
		},
		Nodes: Nodes{
			Text("\n  "),
			Elem{
				Name: Name{Space: `ns_outer`, Local: `four`},
				Attrs: []Attr{
					{Name: Name{Local: `xmlns`}, Value: `ns_outer`},
				},
			},
			Text("\n  "),
			Elem{
				Name: Name{Space: `ns_inner`, Local: `five`},
				Attrs: []Attr{
					{Name: Name{Local: `xmlns`}, Value: `ns_inner`},
					{Name: Name{Local: `six`}, Value: `seven`},
				},
			},
			Text("\n"),
		},
	},
}

func read(t testing.TB, path string) []byte {
	out, err := os.ReadFile(filepath.Join(`test_data`, path))
	require.NoError(t, err)
	return out
}
