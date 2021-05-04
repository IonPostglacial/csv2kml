package csv

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
)

const (
	COL_CN_NAME       = 9
	COL_FAMILY        = 10
	COL_SCI_NAME      = 14
	COL_VARIETY       = 16
	COL_CIRCUMFERENCE = 17
	COL_HEIGHT        = 18
	COL_STAGE         = 19
	COL_COORDS        = 21
)

var iconColor color.NRGBA = color.NRGBA{226, 76, 75, 255}

var palette []color.RGBA = []color.RGBA{
	{24, 77, 71, 0xff},
	{150, 187, 124, 0xff},
	{250, 213, 134, 0xff},
	{198, 71, 86, 0xff},
}

type RecoloredImage struct {
	image.Image
	color color.RGBA
}

func (img *RecoloredImage) At(x, y int) color.Color {
	baseColor := img.Image.At(x, y)
	if x == 13 && y == 40 {
		fmt.Printf("one: %T\n two: %T\ncmp: %+v\n", baseColor, iconColor, baseColor == iconColor)
	}
	if baseColor == iconColor {
		return img.color
	}
	return baseColor
}

//go:embed res/flower.png
var flowerIcon []byte

var InvalidCsvError = errors.New("The CSV file is invalid.")

func ToKml(in io.Reader, out io.Writer) error {
	paletteIndex := 0
	colorsByFamily := map[string]color.RGBA{}
	reader := csv.NewReader(in)
	reader.Comma = ';'
	head := `<?xml version="1.0" encoding="UTF-8"?>
	<kml xmlns="http://www.opengis.net/kml/2.2">
		<Document>
			<name>Les arbres</name>
			<description/>
			<Folder>
				<name>Les arbres</name>`
	tail := `
		</Document>
	</kml>`
	w := zip.NewWriter(out)
	doc, err := w.Create("doc.kml")
	if err != nil {
		return err
	}
	doc.Write([]byte(head))
	if err != nil {
		return err
	}
	for rec, err := reader.Read(); err != io.EOF; rec, err = reader.Read() {
		if len(rec) < 22 {
			return InvalidCsvError
		}
		coords := strings.Split(rec[COL_COORDS], ",")
		if len(coords) != 2 {
			continue
		}
		family := rec[COL_SCI_NAME]
		color, ok := colorsByFamily[family]
		if !ok {
			paletteIndex++
			if paletteIndex >= len(palette) {
				paletteIndex = 0
			}
			color = palette[paletteIndex]
			colorsByFamily[family] = color
		}
		doc.Write([]byte(fmt.Sprintf(
			"<Placemark>"+
				"<name>%s</name>"+
				"<styleUrl>#flower-style-%d</styleUrl>"+
				"<description><![CDATA[<p>stade : %s</p><p>俗名 : %s</p><p>var. %s</p>]]></description>"+
				"<Point>"+
				"<coordinates>%s, %s</coordinates>"+
				"</Point>"+
				"</Placemark>",
			rec[COL_SCI_NAME], paletteIndex, rec[COL_STAGE], rec[COL_CN_NAME], rec[COL_VARIETY], coords[1], coords[0])))
	}
	doc.Write([]byte("</Folder>"))
	for i := 0; i < len(palette); i++ {
		doc.Write([]byte(fmt.Sprintf(`<Style id="flower-style-%d">
              <IconStyle>
                <scale>1</scale>
                <Icon>
                  <href>images/flower-%d.png</href>
                </Icon>
              </IconStyle>
              <LabelStyle>
                <scale>0</scale>
              </LabelStyle>
            </Style>`, i, i)))
	}
	doc.Write([]byte(tail))
	r := bytes.NewReader(flowerIcon)
	img, _, err := image.Decode(r)
	for i, c := range palette {
		imgEntry, err := w.Create(fmt.Sprintf("images/flower-%d.png", i))
		if err != nil {
			return err
		}
		recolored := &RecoloredImage{img, c}
		if err := png.Encode(imgEntry, recolored); err != nil {
			return err
		}
	}
	return w.Close()
}
