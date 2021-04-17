package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	COL_SCI_NAME      = 14
	COL_CIRCUMFERENCE = 17
	COL_HEIGHT        = 18
	COL_STAGE         = 19
	COL_COORDS        = 21
)

func main() {
	input := flag.String("in", "input.csv", "The CSV file to convert.")
	output := flag.String("out", "output.kml", "The KML file to output.")
	in, err := os.Open(*input)
	if err != nil {
		log.Fatalf("Cannot open file '%s'", *input)
	}
	defer in.Close()
	out, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Cannot create file '%s'", *output)
	}
	defer out.Close()
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
			</Folder>
		</Document>
	</kml>`
	out.WriteString(head)
	for rec, err := reader.Read(); err != io.EOF; rec, err = reader.Read() {
		coords := strings.Split(rec[COL_COORDS], ",")
		if len(coords) != 2 {
			continue
		}
		out.WriteString(fmt.Sprintf(
			"<Placemark>"+
				"<name>%s</name>"+
				"<description><![CDATA[<p>stade : %s</p><p>circonférence : %scm</p><p>hauteur : %sm</p>]]></description>"+
				"<Point>"+
				"<coordinates>"+
				"%s, %s"+
				"</coordinates>"+
				"</Point>"+
				"</Placemark>",
			rec[COL_SCI_NAME], rec[COL_STAGE], rec[COL_CIRCUMFERENCE], rec[COL_HEIGHT], coords[1], coords[0]))
	}
	out.WriteString(tail)
}
