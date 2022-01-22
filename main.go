package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"nicolas.galipot.net/csv2kml/csv"
)

const MAX_UPLOAD_SIZE = 10000 * 1024

func serveConverted(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
	file, _, err := r.FormFile("input-file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	defer file.Close()
	var b strings.Builder
	if err := csv.ToKml(file, &b, ';'); err != nil {
		b = strings.Builder{}
		file.Seek(0, io.SeekStart)
		if err = csv.ToKml(file, &b, ','); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<!DOCTYPE html><html><body>Error parsing the CSV: %q <a href='/'>Retry</a></body></html>", err)
			return
		}
	}
	w.Header().Set("Content-Disposition", "attachment; filename=output.kmz")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	io.Copy(w, strings.NewReader(b.String()))
}

func serveGui(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><body><form method="POST" action="/convert" enctype="multipart/form-data">File to convert<input type="file" name="input-file"><button>Convert</button></form></body></html>`)
}

func main() {
	serveFs := flag.NewFlagSet("serve", flag.ExitOnError)
	host := serveFs.String("host", "localhost", "The host name.")
	port := serveFs.String("port", "8080", "The port to listen.")
	if os.Args[1] == "serve" {
		serveFs.Parse(os.Args[2:])
		http.HandleFunc("/convert", serveConverted)
		http.HandleFunc("/", serveGui)
		http.ListenAndServe(*host+":"+*port, nil)
		return
	}
	input := flag.String("in", "input.csv", "The CSV file to convert.")
	output := flag.String("out", "output.kmz", "The KMZ file to output.")
	comma := flag.String("sep", ";", "The CSV separator.")
	flag.Parse()
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
	r, size := utf8.DecodeRuneInString(*comma)
	if size != 1 {
		log.Fatal("The sep argument should contain a single character.")
	}
	if err := csv.ToKml(in, out, r); err != nil {
		log.Fatal(err)
	}
}
