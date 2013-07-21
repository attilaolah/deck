// pdfdeck: make PDF slide decks
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"bitbucket.org/zombiezen/gopdf/pdf"
	"github.com/ajstarks/deck"
)

// fontmap maps generic font names to specific implementation names
var fontmap = map[string]string{
	"sans": pdf.Helvetica, "sans-bold": pdf.HelveticaBold, "sans-italic": pdf.HelveticaOblique,
	"serif": pdf.Times, "serif-bold": pdf.TimesBold, "serif-italic": pdf.TimesItalic,
	"mono": pdf.Courier, "mono-bold": pdf.CourierBold, "mono-italic": pdf.CourierOblique,
}

// grid makes a grid using a percentage scale
func grid(c *pdf.Canvas, w, h pdf.Unit, pct float64) {
	p1 := pdf.Point{X: 0, Y: 0}
	p2 := pdf.Point{X: 0, Y: h}
	pw := w * (pdf.Unit(pct) / 100)
	ph := h * (pdf.Unit(pct) / 100)
	c.Push()
	c.SetStrokeColor(0.9, 0.9, 0.9)
	c.SetLineWidth(1)
	for x := pdf.Unit(0.0); x <= w; x += pw {
		p1.X = x
		p2.X = x
		c.DrawLine(p1, p2)
	}
	p1.X = 0
	p2.X = w
	for y := pdf.Unit(0.0); y <= h; y += ph {
		p1.Y = y
		p2.Y = y
		c.DrawLine(p1, p2)
	}
	c.Pop()
}

// bullet draws a rectangular bullet
func bullet(c *pdf.Canvas, x, y, size pdf.Unit, color string) {
	rs := size / 2
	path := new(pdf.Path)
	rect := pdf.Rectangle{Min: pdf.Point{X: x, Y: y}, Max: pdf.Point{X: x + rs, Y: y + rs}}
	path.Rectangle(rect)
	c.Fill(path)
}

// background places a colored rectangle
func background(c *pdf.Canvas, w, h pdf.Unit, color string) {
	path := new(pdf.Path)
	rect := pdf.Rectangle{Min: pdf.Point{X: 0, Y: 0}, Max: pdf.Point{X: w, Y: h}}
	path.Rectangle(rect)
	r, g, b := colorlookup(color)
	c.SetColor(r, g, b)
	c.Fill(path)
}

// dotext places text elements on the canvas according to type
func dotext(c *pdf.Canvas, cw, x, y, fs pdf.Unit, wp float64, tdata, font, color, align, ttype string) {
	var tw pdf.Unit

	td := strings.Split(tdata, "\n")
	red, green, blue := colorlookup(color)
	if ttype == "code" {
		font = "mono"
		c.Push()
		ch := pdf.Unit(len(td)) * (pdf.Unit(1.8) * fs)
		c.Translate(x-fs, (y-ch)+fs)
		tw = pdf.Unit(deck.Pwidth(wp, float64(cw), float64(cw-x-20)))
		background(c, tw, ch, "rgb(240,240,240)")
		c.Pop()
	}
	c.Push()
	c.SetColor(red, green, blue)
	if ttype == "block" {
		tw = pdf.Unit(deck.Pwidth(wp, float64(cw), float64(cw/2)))
		textwrap(c, x, y, tw, fs, fs*1.4, tdata, font)
	} else {
		ls := pdf.Unit(1.8) * fs
		for _, t := range td {
			showtext(c, x, y, t, fs, font, align)
			y -= ls
		}
	}
	c.Pop()
}

// showtext places fully attributed text at the specified location
func showtext(c *pdf.Canvas, x, y pdf.Unit, s string, fs pdf.Unit, font, align string) {
	var offset pdf.Unit = 0
	text := new(pdf.Text)
	c.Push()
	text.SetFont(fontlookup(font), fs)
	text.Text(s)
	tw := text.X()
	switch align {
	case "center":
		offset = -(tw / 2)
	case "right":
		offset = -tw
	}
	c.Translate(x+offset, y)
	c.DrawText(text)
	c.Pop()
}

// dolists places lists on the canvas
func dolist(c *pdf.Canvas, x, y, fs pdf.Unit, tdata []string, font, color, ltype string) {
	c.Push()
	text := new(pdf.Text)
	if font == "" {
		font = "sans"
	}
	text.SetFont(fontlookup(font), fs)
	red, green, blue := colorlookup(color)
	c.SetColor(red, green, blue)
	if ltype == "bullet" {
		x += fs
	}
	c.Translate(x, y)
	ls := pdf.Unit(2.0) * fs
	for i, t := range tdata {
		if ltype == "number" {
			t = fmt.Sprintf("%d. ", i+1) + t
		}
		if ltype == "bullet" {
			bullet(c, -fs, fs/8, fs, color)
		}
		text.Text(t)
		c.DrawText(text)
		c.Translate(0, -ls)
	}
	c.Pop()
}

// doimage places images on the canvas
func doimage(c *pdf.Canvas, x, y pdf.Unit, width, height int, name string) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	iw, ih := pdf.Unit(width), pdf.Unit(height)
	r := pdf.Rectangle{Min: pdf.Point{X: 0, Y: 0}, Max: pdf.Point{X: iw, Y: ih}}
	c.Push()
	c.Translate(x-(iw/2), y-(ih/2)) // center
	c.DrawImage(img, r)
	c.Pop()
}

// whitespace determines if a rune is whitespace
func whitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t'
}

// fontlookup maps font aliases to implementation font names
func fontlookup(s string) string {
	font, ok := fontmap[s]
	if ok {
		return font
	}
	return "sans"
}

// textwrap draws text at location, wrapping at the specified width
func textwrap(c *pdf.Canvas, x, y, w, fs, leading pdf.Unit, s, font string) {
	text := new(pdf.Text)
	text.SetFont(fontlookup(font), fs)
	words := strings.FieldsFunc(s, whitespace)
	edge := (x + w) * 0.75
	c.Push()
	c.Translate(x, y)
	for _, s := range words {
		text.Text(s + " ")
		tx := text.X()
		if tx > edge {
			text.NextLine()
			c.DrawText(text)
			c.Translate(0, -leading)
		}
	}
	c.DrawText(text)
	c.Pop()
}

// pct converts percentages to canvas measures
func pct(p float64, m pdf.Unit) pdf.Unit {
	return pdf.Unit((p / 100.0) * float64(m))
}

// dcoord returns coordinates in canvas units
func dcoord(xp, yp float64, w, h pdf.Unit) (x, y pdf.Unit) {
	x = pct(xp, w)
	y = pct(yp, h)
	return
}

// dimen returns location and size based on canvas dimensions
func dimen(d deck.Deck, xp, yp, sp float64) (x, y, s pdf.Unit) {

	c := d.Canvas
	xf, yf, sf := deck.Dimen(c, xp, yp, sp)
	x, y, s = pdf.Unit(xf), pdf.Unit(yf), pdf.Unit(sf)
	return
}

// doslides reads the deck file, making the PDF version
func doslides(doc *pdf.Document, filename string, w, h int, gp float64) {
	var d deck.Deck
	var err error

	d, err = deck.Read(filename, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	if d.Canvas.Width == 0 {
		d.Canvas.Width = int(pdf.USLetterHeight) // landscape
	}
	if d.Canvas.Height == 0 {
		d.Canvas.Height = int(pdf.USLetterWidth) // landscape
	}

	for i := 0; i < len(d.Slide); i++ {
		pdfslide(doc, d, i, gp)
	}
}

// pdfslide makes a slide, one slide per PDF page
func pdfslide(doc *pdf.Document, d deck.Deck, n int, gp float64) {
	if n < 0 || n > len(d.Slide)-1 {
		return
	}
	var x, y, fs pdf.Unit
	var canvas *pdf.Canvas

	cw := pdf.Unit(d.Canvas.Width)
	ch := pdf.Unit(d.Canvas.Height)
	canvas = doc.NewPage(cw, ch)
	slide := d.Slide[n]

	// set background, if specified
	if len(slide.Bg) > 0 {
		background(canvas, cw, ch, slide.Bg)
	}
	// set the default foreground
	if slide.Fg == "" {
		slide.Fg = "black"
	}
	if gp > 0 {
		grid(canvas, cw, ch, gp)
	}
	// for every image on the slide...
	for _, im := range slide.Image {
		x, y = dcoord(im.Xp, im.Yp, cw, ch)
		doimage(canvas, x, y, im.Width, im.Height, im.Name)
	}
	// for every text element...
	for _, t := range slide.Text {
		if t.Color == "" {
			t.Color = slide.Fg
		}
		if t.Font == "" {
			t.Font = "sans"
		}
		x, y, fs = dimen(d, t.Xp, t.Yp, t.Sp)
		dotext(canvas, cw, x, y, fs, t.Wp, t.Tdata, t.Font, t.Color, t.Align, t.Type)
	}
	// for every list element...
	for _, l := range slide.List {
		if l.Color == "" {
			l.Color = slide.Fg
		}
		x, y, fs = dimen(d, l.Xp, l.Yp, l.Sp)
		dolist(canvas, x, y, fs, l.Li, l.Font, l.Color, l.Type)
	}
	canvas.Close()
}

// dodeck kicks things off
func dodeck(filename string, gp float64) {
	doc := pdf.New()
	doslides(doc, filename, int(pdf.USLetterHeight), int(pdf.USLetterWidth), gp)
	err := doc.Encode(os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// for every file, make a deck
func main() {
	var gridpct = flag.Float64("g", 0, "place percentage grid on each slide")
	flag.Parse()
	for _, f := range flag.Args() {
		dodeck(f, *gridpct)
	}
}