//go:build wasm

package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"syscall/js"
)

type attachment struct {
	name   string
	typ    string
	data   []byte
	isBody bool
}

func extractAttachments(emlData []byte) []attachment {
	msg, err := mail.ReadMessage(bytes.NewReader(emlData))
	if err != nil {
		return nil
	}

	_, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		return nil
	}

	var attachments []attachment
	processParts(multipart.NewReader(msg.Body, params["boundary"]), &attachments)
	return attachments
}

func processParts(mr *multipart.Reader, attachments *[]attachment) {
	for {
		part, err := mr.NextPart()
		if err != nil {
			return
		}

		name := part.FileName()
		if name == "" {
			if _, params, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition")); params != nil {
				name = params["filename"]
			}
		}

		ct := part.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/") {
			if _, params, _ := mime.ParseMediaType(ct); params != nil {
				if b := params["boundary"]; b != "" {
					processParts(multipart.NewReader(part, b), attachments)
				}
			}
			continue
		}

		if data, _ := io.ReadAll(decodeBody(part.Header, part)); data != nil {
			if name != "" {
				*attachments = append(*attachments, attachment{name, ct, data, false})
			} else if strings.HasPrefix(ct, "text/html") {
				*attachments = append(*attachments, attachment{"", ct, data, true})
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

func extractJS(this js.Value, args []js.Value) interface{} {
	if len(args) == 0 {
		return js.Global().Get("Array").New()
	}

	emlData := make([]byte, args[0].Length())
	js.CopyBytesToGo(emlData, args[0])

	result := js.Global().Get("Array").New()
	for _, att := range extractAttachments(emlData) {
		obj := js.Global().Get("Object").New()
		obj.Set("name", att.name)
		obj.Set("type", att.typ)
		obj.Set("isBody", att.isBody)
		uint8 := js.Global().Get("Uint8Array").New(len(att.data))
		js.CopyBytesToJS(uint8, att.data)
		obj.Set("data", uint8)
		result.Call("push", obj)
	}
	return result
}

func main() {
	js.Global().Set("extractAttachments", js.FuncOf(extractJS))
	select {}
}
