# http-proxy-firewall

### Idea

Layer 7 Proxy Firewall.

**Warning**: clone it and adapt for own use cases

---

### Deployment

---
#### env var:
This file is for sensitive variables (like keys, secrets and etc).

Make a copy of .env.example to .env
```shell
cp .env.example .env
```
... and define required variables

---

#### proxy-firewall.conf:
This file is for service specific non sensitive configs (like flags, addresses and etc).

After building and copying to `/etc/proxy-firewall/proxy-firewall.conf` check for necessary parameters before starting as systemd service.

---

#### unit file in:
```
/usr/lib/systemd/system
```

```shell
mkdir -p /etc/proxy-firewall/bin
mkdir -p /etc/proxy-firewall/files
mkdir -p /etc/proxy-firewall/log
rm /etc/proxy-firewall/log/*
go build -o /etc/proxy-firewall/bin/proxy-firewall main.go
chmod +x /etc/proxy-firewall/bin/proxy-firewall
cp proxy-firewall.conf /etc/proxy-firewall/proxy-firewall.conf
cp .env /etc/proxy-firewall/.env

cp proxy-firewall.service /usr/lib/systemd/system/
systemctl daemon-reload
systemctl enable proxy-firewall
systemctl restart proxy-firewall
systemctl status proxy-firewall
```

```shell
# to cleanup previous deploy
systemctl stop proxy-firewall
systemctl disable proxy-firewall
pkill -f proxy-firewall
```
