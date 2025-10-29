//go:build !wasm

package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	file, err := os.Open("message.eml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	msg, err := mail.ReadMessage(file)
	if err != nil {
		panic(err)
	}

	fmt.Println("From:", msg.Header.Get("From"))
	fmt.Println("To:", msg.Header.Get("To"))
	fmt.Println("Subject:", msg.Header.Get("Subject"))

	contentType := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		fmt.Println("Not multipart — showing body:")
		body, _ := io.ReadAll(decodeBody(msg.Header, msg.Body))
		fmt.Println(string(body))
		return
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	processParts(mr, "./attachments")
}

func processParts(mr *multipart.Reader, outDir string) {
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading part:", err)
			break
		}

		ct := part.Header.Get("Content-Type")
		cd := part.Header.Get("Content-Disposition")
		name := part.FileName()

		if name == "" {
			_, params, _ := mime.ParseMediaType(cd)
			name = params["filename"]
		}

		if strings.HasPrefix(ct, "multipart/") {
			// Nested multipart (e.g. alternative inside mixed)
			_, params, _ := mime.ParseMediaType(ct)
			if b, ok := params["boundary"]; ok {
				fmt.Println("→ Nested multipart:", ct)
				nested := multipart.NewReader(part, b)
				processParts(nested, outDir)
			}
			continue
		}

		data, _ := io.ReadAll(decodeBody(part.Header, part))

		os.MkdirAll(outDir, 0755)
		if name != "" {
			safeName := filepath.Base(name)
			if err := os.WriteFile(filepath.Join(outDir, safeName), data, 0644); err != nil {
				fmt.Println("❌ Error saving", safeName, ":", err)
			} else {
				fmt.Printf("📎 Saved attachment: %s (%s)\n", safeName, ct)
			}
		} else if strings.HasPrefix(ct, "text/html") {
			if err := os.WriteFile(filepath.Join(outDir, "body.html"), data, 0644); err != nil {
				fmt.Println("❌ Error saving body.html:", err)
			} else {
				fmt.Println("📧 Saved HTML body: body.html")
			}
		}
	}
}

type headerGetter interface {
	Get(string) string
}

func decodeBody(header headerGetter, r io.Reader) io.Reader {
	encoding := strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding")))
	switch encoding {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, r)
	case "quoted-printable":
		return quotedprintable.NewReader(r)
	default:
		return r
	}
}
