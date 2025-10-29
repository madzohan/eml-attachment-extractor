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
		if name == "" {
			_, params, _ := mime.ParseMediaType(ct)
			name = params["name"]
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
		} else if strings.HasPrefix(ct, "text/plain") {
			if err := os.WriteFile(filepath.Join(outDir, "body.txt"), data, 0644); err != nil {
				fmt.Println("❌ Error saving body.txt:", err)
			} else {
				fmt.Println("📧 Saved text body: body.txt")
			}
		} else if cd != "" || part.Header.Get("Content-ID") != "" {
			// Inline attachment without filename
			ext := ".bin"
			if strings.HasPrefix(ct, "text/plain") {
				ext = ".txt"
			} else if strings.HasPrefix(ct, "image/") {
				ext = strings.TrimPrefix(strings.Split(ct, ";")[0], "image/")
				ext = "." + ext
			}
			cid := strings.Trim(part.Header.Get("Content-ID"), "<>")
			if cid == "" {
				cid = "inline"
			}
			safeName := filepath.Base(cid) + ext
			if err := os.WriteFile(filepath.Join(outDir, safeName), data, 0644); err != nil {
				fmt.Println("❌ Error saving", safeName, ":", err)
			} else {
				fmt.Printf("📎 Saved inline: %s (%s)\n", safeName, ct)
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
