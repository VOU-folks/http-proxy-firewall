# http-proxy-firewall

### Idea

Layer 7 Proxy Firewall.

**Warning**: clone it and adapt for own use cases

---

### Deployment

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

# for cleanup previous deploy
systemctl disable proxy-firewall
systemctl daemon-reload
pkill -f proxy-firewall

cp proxy-firewall.service /usr/lib/systemd/system/
systemctl daemon-reload
systemctl enable proxy-firewall
systemctl restart proxy-firewall
systemctl status proxy-firewall
```

