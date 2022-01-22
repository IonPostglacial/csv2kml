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
	ColNameCn = iota
	ColNameSci
	ColVariety
	ColStage
	ColCoords
	ColNumber
)

var columnNames = []string{
	"俗名",
	"Nom scientifique",
	"VARIETE OUCULTIVAR",
	"STADE DE DEVELOPPEMENT",
	"geo_point_2d",
}

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
	if baseColor == iconColor {
		return img.color
	}
	return baseColor
}

//go:embed res/flower.png
var flowerIcon []byte

var ErrInvalidCsv = errors.New("the CSV file is invalid")

func ToKml(in io.Reader, out io.Writer, comma rune) error {
	paletteIndex := 0
	colorsByFamily := map[string]color.RGBA{}
	reader := csv.NewReader(in)
	reader.Comma = comma
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
	isHeader := true
	colIndices := make([]int, ColNumber)
	for i := range colIndices {
		colIndices[i] = -1
	}
	if err != nil {
		return err
	}
	doc.Write([]byte(head))
	if err != nil {
		return err
	}
	for rec, err := reader.Read(); err != io.EOF; rec, err = reader.Read() {
		if isHeader {
			for i, colName := range rec {
				switch colName {
				case columnNames[ColNameCn]:
					colIndices[ColNameCn] = i
				case columnNames[ColNameSci]:
					colIndices[ColNameSci] = i
				case columnNames[ColVariety]:
					colIndices[ColVariety] = i
				case columnNames[ColStage]:
					colIndices[ColStage] = i
				case columnNames[ColCoords]:
					colIndices[ColCoords] = i
				}
			}
			missingColumns := make([]string, 0, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				if colIndices[i] < 0 {
					missingColumns = append(missingColumns, columnNames[i])
				}
			}
			if len(missingColumns) > 0 {
				return fmt.Errorf("some columns are missing: %s", strings.Join(missingColumns, ", "))
			}
			isHeader = false
		}
		coords := strings.Split(rec[colIndices[ColCoords]], ",")
		if len(coords) != 2 {
			continue
		}
		family := rec[colIndices[ColNameSci]]
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
			rec[colIndices[ColNameSci]], paletteIndex, rec[colIndices[ColStage]], rec[colIndices[ColNameCn]], rec[colIndices[ColVariety]], coords[1], coords[0])))
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
	if err != nil {
		return err
	}
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
