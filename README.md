# EML Attachment Extractor

🚀 Extract attachments and HTML body from `.eml` files directly in your browser

**[Try it live](https://madzohan.github.io/eml-attachment-extractor/)**

## Tech Stack

- **TinyGo** → compiles Go to WebAssembly
- **WASM** → runs in browser without backend
- **GitHub Pages** → auto-deploy on push

## How It Works

1. User selects `.eml` file
2. JavaScript reads file as bytes
3. Passes data to WASM module
4. Go parses email (MIME multipart, base64, quoted-printable)
5. Extracts attachments and HTML body
6. Creates download links in browser

## Local Development

```bash
# Build WASM
make wasm

# Run local server
make serve

# Open http://localhost:8080
```

## Deploy

Push to `main` branch → GitHub Actions builds and deploys automatically