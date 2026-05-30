import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';

const PORT = parseInt(process.env.PORT || '5173', 10);
const HOST = process.env.HOST || '0.0.0.0';
const BUILD_DIR = path.join(process.cwd(), 'build');

const MIME = {
	'.html': 'text/html',
	'.js': 'application/javascript',
	'.mjs': 'application/javascript',
	'.css': 'text/css',
	'.json': 'application/json',
	'.svg': 'image/svg+xml',
	'.ico': 'image/x-icon',
	'.txt': 'text/plain',
};

function serveFile(res, filePath, mime) {
	fs.readFile(filePath, (err, data) => {
		if (err) {
			res.writeHead(404);
			res.end('Not found');
			return;
		}
		res.writeHead(200, { 'Content-Type': mime });
		res.end(data);
	});
}

const server = http.createServer((req, res) => {
	let url = req.url || '/';

	// Strip query strings
	url = url.split('?')[0];

	// Default to index.html for SPA routing
	let filePath = path.join(BUILD_DIR, url);
	const ext = path.extname(url);

	if (fs.existsSync(filePath)) {
		const mime = MIME[ext] || 'application/octet-stream';
		serveFile(res, filePath, mime);
	} else {
		// Fallback to index.html for SPA routing
		const indexPath = path.join(BUILD_DIR, 'index.html');
		if (fs.existsSync(indexPath)) {
			return serveFile(res, indexPath, 'text/html');
		}
		res.writeHead(404);
		res.end('Not found');
	}
});

server.listen(PORT, HOST, () => {
	console.log(`Frontend server listening on ${HOST}:${PORT}`);
});