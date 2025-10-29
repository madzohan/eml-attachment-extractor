#!/usr/bin/env python3
import http.server
import socketserver

class WasmHandler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header('Cross-Origin-Opener-Policy', 'same-origin')
        self.send_header('Cross-Origin-Embedder-Policy', 'require-corp')
        super().end_headers()
    
    def guess_type(self, path):
        mimetype = super().guess_type(path)
        if path.endswith('.wasm'):
            return 'application/wasm'
        return mimetype

PORT = 8080
try:
    with socketserver.TCPServer(("", PORT), WasmHandler) as httpd:
        print(f"Server running at http://localhost:{PORT}/")
        httpd.serve_forever()
except KeyboardInterrupt:
    print("\nServer stopped")
