# Build stage
FROM node:25-alpine AS builder

WORKDIR /app

COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm@10 && pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

# Runtime stage — static SPA server
FROM node:25-alpine

WORKDIR /app

COPY --from=builder /app/build ./build

ENV HOST=0.0.0.0
ENV PORT=5173

EXPOSE 5173

CMD ["node", "-e", "const http=require('http'),fs=require('fs'),path=require('path');const m={'.html':'text/html','.js':'application/javascript','.css':'text/css','.json':'application/json','.svg':'image/svg+xml'};http.createServer((r,s)=>{let u=r.url||'/';u=u.split('?')[0];let f=path.join('/app/build',u);if(fs.existsSync(f)&&fs.statSync(f).isDirectory())f=path.join(f,'index.html');if(!path.extname(f)){const i=path.join('/app/build','index.html');if(fs.existsSync(i))f=i}const e=path.extname(f);if(fs.existsSync(f)){s.writeHead(200,{'Content-Type':m[e]||'application/octet-stream'});s.end(fs.readFileSync(f))}else{s.writeHead(404);s.end('Not found')}}).listen(5173,'0.0.0.0',()=>console.log(' listening'));"]