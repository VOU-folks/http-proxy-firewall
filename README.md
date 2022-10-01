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
mkdir -p /etc/proxy-firewall/log
go build -o /etc/proxy-firewall/bin/proxy-firewall main.go
chmod +x /etc/proxy-firewall/bin/proxy-firewall

# for cleanup previous deploy
systemctl disable proxy-firewall
systemctl daemon-reload
pkill -f proxy-firewall

cp proxy-firewall.service /usr/lib/systemd/system/
systemctl enable proxy-firewall
systemctl start proxy-firewall
systemctl status proxy-firewall
```

