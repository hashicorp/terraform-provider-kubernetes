package mdplain

import (
	"bytes"

	"github.com/russross/blackfriday"
)

type Text struct{}

func TextRenderer() blackfriday.Renderer {
	return &Text{}
}

func (options *Text) GetFlags() int {
	return 0
}

func (options *Text) TitleBlock(out *bytes.Buffer, text []byte) {
	text = bytes.TrimPrefix(text, []byte("% "))
	text = bytes.Replace(text, []byte("\n% "), []byte("\n"), -1)
	out.Write(text)
	out.WriteString("\n")
}

func (options *Text) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	marker := out.Len()
	doubleSpace(out)

	if !text() {
		out.Truncate(marker)
		return
	}
}

func (options *Text) BlockHtml(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.Write(text)
	out.WriteByte('\n')
}

func (options *Text) HRule(out *bytes.Buffer) {
	doubleSpace(out)
}

func (options *Text) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	options.BlockCodeNormal(out, text, lang)
}

func (options *Text) BlockCodeNormal(out *bytes.Buffer, text []byte, lang string) {
	doubleSpace(out)
	out.Write(text)
}

func (options *Text) BlockQuote(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.Write(text)
}

func (options *Text) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {
	doubleSpace(out)
	out.Write(header)
	out.Write(body)
}

func (options *Text) TableRow(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.Write(text)
}

func (options *Text) TableHeaderCell(out *bytes.Buffer, text []byte, align int) {
	doubleSpace(out)
	out.Write(text)
}

func (options *Text) TableCell(out *bytes.Buffer, text []byte, align int) {
	doubleSpace(out)
	out.Write(text)
}

func (options *Text) Footnotes(out *bytes.Buffer, text func() bool) {
	options.HRule(out)
	options.List(out, text, 0)
}

func (options *Text) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int) {
	out.Write(text)
}

func (options *Text) List(out *bytes.Buffer, text func() bool, flags int) {
	marker := out.Len()
	doubleSpace(out)

	if !text() {
		out.Truncate(marker)
		return
	}
}

func (options *Text) ListItem(out *bytes.Buffer, text []byte, flags int) {
	out.Write(text)
}

func (options *Text) Paragraph(out *bytes.Buffer, text func() bool) {
	marker := out.Len()
	doubleSpace(out)

	if !text() {
		out.Truncate(marker)
		return
	}
}

func (options *Text) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	out.Write(link)
}

func (options *Text) CodeSpan(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *Text) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *Text) Emphasis(out *bytes.Buffer, text []byte) {
	if len(text) == 0 {
		return
	}
	out.Write(text)
}

func (options *Text) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	return
}

func (options *Text) LineBreak(out *bytes.Buffer) {
	return
}

func (options *Text) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	out.Write(content)
	if !isRelativeLink(link) {
		out.WriteString(" ")
		out.Write(link)
	}
	return
}

func (options *Text) RawHtmlTag(out *bytes.Buffer, text []byte) {
	return
}

func (options *Text) TripleEmphasis(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *Text) StrikeThrough(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *Text) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
	return
}

func (options *Text) Entity(out *bytes.Buffer, entity []byte) {
	out.Write(entity)
}

func (options *Text) NormalText(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (options *Text) Smartypants(out *bytes.Buffer, text []byte) {
	return
}

func (options *Text) DocumentHeader(out *bytes.Buffer) {
	return
}

func (options *Text) DocumentFooter(out *bytes.Buffer) {
	return
}

func (options *Text) TocHeader(text []byte, level int) {
	return
}

func (options *Text) TocFinalize() {
	return
}

func doubleSpace(out *bytes.Buffer) {
	if out.Len() > 0 {
		out.WriteByte('\n')
	}
}

func isRelativeLink(link []byte) (yes bool) {
	yes = false

	// a tag begin with '#'
	if link[0] == '#' {
		yes = true
	}

	// link begin with '/' but not '//', the second maybe a protocol relative link
	if len(link) >= 2 && link[0] == '/' && link[1] != '/' {
		yes = true
	}

	// only the root '/'
	if len(link) == 1 && link[0] == '/' {
		yes = true
	}
	return
}
