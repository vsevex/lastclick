# Manual Deployment — Hetzner Server

Build locally on Windows, upload artifacts, run natively on the server. No Docker.

---

## 1. Local Build

### Backend (Go → Linux binary)

```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/lastclick ./cmd/server
```

### Frontend (Vite SPA)

```bash
cd web
VITE_WS_URL="wss://lastclick.vsevex.me" npm run build
cd ..
```

This produces `web/dist/` with static files (HTML, JS, CSS).

### Install goose locally (one-time)

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/goose github.com/pressly/goose/v3/cmd/goose
```

---

## 2. Upload to Server

```bash
scp bin/lastclick root@<server-ip>:/opt/lastclick/
scp bin/goose root@<server-ip>:/opt/lastclick/
scp -r migrations root@<server-ip>:/opt/lastclick/
scp -r web/dist/* root@<server-ip>:/opt/lastclick/web/dist/
scp nginx/default.conf root@<server-ip>:/etc/nginx/conf.d/lastclick.conf
```

Directory layout on server:

```bash
/opt/lastclick/
├── lastclick              # Go binary
├── goose                  # migration tool
├── migrations/            # SQL files
└── web/
    └── dist/              # Vite build output (static SPA)
        ├── index.html
        └── assets/
```

---

## 3. Server Prerequisites

```bash
apt update && apt upgrade -y
apt install -y nginx postgresql redis-server
systemctl enable --now postgresql redis-server nginx
```

### PostgreSQL

```bash
sudo -u postgres psql <<SQL
CREATE USER vsevex WITH PASSWORD '1596225600';
CREATE DATABASE lastclick OWNER vsevex;
SQL
```

### Run Migrations

```bash
cd /opt/lastclick
chmod +x lastclick goose
./goose -dir migrations postgres "postgres://vsevex:1596225600@localhost:5432/lastclick?sslmode=disable" up
```

---

## 4. Environment File

```bash
cat > /opt/lastclick/.env << 'EOF'
ENV=production
HTTP_ADDR=:8080
BOT_TOKEN=<your-bot-token>
MINI_APP_URL=https://lastclick.vsevex.me
DATABASE_URL=postgres://vsevex:1596225600@localhost:5432/lastclick?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
EOF

chmod 600 /opt/lastclick/.env
```

---

## 5. Systemd Services

### Backend

```bash
cat > /etc/systemd/system/lastclick.service << 'EOF'
[Unit]
Description=LastClick Backend
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
WorkingDirectory=/opt/lastclick
EnvironmentFile=/opt/lastclick/.env
ExecStart=/opt/lastclick/bin/lastclick
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

### Enable and start

The frontend is a static SPA served by nginx — no systemd service needed.

```bash
systemctl daemon-reload
systemctl enable --now lastclick
```

---

## 6. TLS Certificates

```bash
apt install -y certbot python3-certbot-nginx
certbot --nginx -d lastclick.vsevex.me
```

Certbot auto-configures nginx SSL and sets up renewal.

---

## 7. Nginx Config

Replace `/etc/nginx/conf.d/lastclick.conf` (or copy `nginx/default.conf` from the repo):

```nginx
upstream backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}

server {
    listen 80;
    server_name lastclick.vsevex.me;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name lastclick.vsevex.me;

    ssl_certificate     /etc/letsencrypt/live/lastclick.vsevex.me/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/lastclick.vsevex.me/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:10m;
    ssl_session_tickets off;
    ssl_ciphers HIGH:!aNULL:!MD5;

    client_max_body_size 10m;

    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    add_header Referrer-Policy no-referrer-when-downgrade;

    location /ws {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    location /api/ {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://backend;
    }

    location /metrics {
        proxy_pass http://backend;
    }

    location /assets/ {
        root /opt/lastclick/web/dist;
        expires 365d;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    location / {
        root /opt/lastclick/web/dist;
        try_files $uri $uri/ /index.html;
    }
}
```

```bash
nginx -t && systemctl reload nginx
```

---

## 8. Telegram Webhook

```bash
curl "https://api.telegram.org/bot<your-bot-token>/setWebhook?url=https://lastclick.vsevex.me"
```

---

## 9. Redeploy Workflow

Run locally after code changes:

```bash
# Build
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/lastclick ./cmd/server
cd web && VITE_WS_URL="wss://lastclick.vsevex.me" npm run build && cd ..

# Upload
scp bin/lastclick root@<server-ip>:/opt/lastclick/
scp -r web/dist/* root@<server-ip>:/opt/lastclick/web/dist/

# Restart backend + reload nginx
ssh root@<server-ip> "systemctl restart lastclick && nginx -t && systemctl reload nginx"
```

---

## 10. Useful Commands

```bash
# Logs
journalctl -u lastclick -f

# Status
systemctl status lastclick

# Run migrations
cd /opt/lastclick && ./goose -dir migrations postgres "$DATABASE_URL" up

# Rollback last migration
cd /opt/lastclick && ./goose -dir migrations postgres "$DATABASE_URL" down
```
