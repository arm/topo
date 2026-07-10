#!/usr/bin/env python3
import os
from http.server import HTTPServer, BaseHTTPRequestHandler

class HelloHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        name = os.environ.get('NAME', 'World')
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.wfile.write(f'Hello {name}\n'.encode())

    def log_message(self, format, *args):
        pass

if __name__ == '__main__':
    port = int(os.environ.get('PORT', '8080'))
    server = HTTPServer(('0.0.0.0', port), HelloHandler)
    print(f'Server listening on port {port}')
    server.serve_forever()
