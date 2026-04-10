// Command smlreader reads SML binary data and prints meter readings in the
// same format as the C sml_server reference tool (OBIS#value#unit).
//
// Usage:
//
//	smlreader <file>    # read from file
//	smlreader -         # read from stdin
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/databus23/go-sml"
	"github.com/databus23/go-sml/transport"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <file|->\n", os.Args[0])
		os.Exit(1)
	}

	var r io.Reader
	if os.Args[1] == "-" {
		r = os.Stdin
	} else {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		r = bytes.NewReader(data)
	}

	reader := transport.NewReader(r)
	for {
		frame, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		file, _ := sml.DecodeWithOptions(frame, sml.DecodeOptions{Strict: false})
		for _, msg := range file.Messages {
			glr, ok := msg.Body.(*sml.GetListResponse)
			if !ok {
				continue
			}
			for i := range glr.ValList {
				line, ok := sml.FormatReadingValue(&glr.ValList[i])
				if ok {
					fmt.Println(line)
				}
			}
		}
	}
}
