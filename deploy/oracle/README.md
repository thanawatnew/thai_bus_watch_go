# Oracle Cloud Always Free deployment

This deployment runs Thai Bus Watch behind Caddy with automatic HTTPS. Both
containers restart after a crash or VM reboot. Telegram is disabled when its
variables are blank.

## 1. Create the VM

In the Oracle Cloud Console, create an **Always Free eligible** Compute instance:

- Image: Ubuntu 24.04 (Canonical)
- Shape: `VM.Standard.A1.Flex`
- Size: 1 OCPU and 6 GB RAM
- Networking: public subnet and a public IPv4 address
- SSH: upload your own public key or download Oracle's generated key pair

Reserve the public IP after creation so a stop/start does not change DNS.

Add stateful ingress rules to the instance's Network Security Group (preferred)
or subnet security list:

- TCP 22 from your own public IP only
- TCP 80 from `0.0.0.0/0`
- TCP 443 from `0.0.0.0/0`
- UDP 443 from `0.0.0.0/0` (optional HTTP/3)

Do not expose port 8080 publicly; only Caddy should reach it.

## 2. Point DNS at the VM

Create an `A` record such as `bus.example.com` pointing to the reserved public
IPv4 address. Wait until the record resolves before starting Caddy.

## 3. Install and deploy

SSH to the VM as `ubuntu`, then run:

```sh
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-v2 git
sudo systemctl enable --now docker
sudo usermod -aG docker ubuntu
exit
```

Reconnect so the Docker group applies, then run:

```sh
git clone https://github.com/thanawatnew/thai_bus_watch_go.git
cd thai_bus_watch_go/deploy/oracle
cp .env.example .env
nano .env
docker compose up -d --build
docker compose ps
curl -fsS https://YOUR_DOMAIN/healthz
```

Set `DOMAIN` in `.env`. Leave both Telegram values blank for a map-only server.
Never commit `.env`.

## Updating

```sh
cd ~/thai_bus_watch_go
git pull --ff-only
cd deploy/oracle
docker compose up -d --build
docker image prune -f
```

## Diagnostics and backup

```sh
docker compose ps
docker compose logs --tail=100 buswatch
docker compose logs --tail=100 caddy
curl -fsS http://127.0.0.1:8080/healthz
```

Caddy's named volumes contain its TLS state. The application itself currently
stores active watches in memory, so they are cleared by a container restart.
