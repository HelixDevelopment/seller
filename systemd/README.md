# Helix Seller — systemd User Services

## Services

| Unit | Description |
|------|-------------|
| `helix.target` | Meta-target that starts/stops the full platform stack |
| `helix-postgres.service` | PostgreSQL 16 (via Podman) |
| `container-helix-postgres.service` | Auto-generated Podman service for PostgreSQL |
| `container-helix-redis.service` | Auto-generated Podman service for Redis |
| `container-helix-nats.service` | NATS with JetStream (via Podman) |
| `helix-server.service` | Go server binary (`build/helix-server`) |
| `helix-angular.service` | Angular dev server (`web/`, via `npm start`) — optional |

## Install

```bash
cp systemd/*.service ~/.config/systemd/user/ && systemctl --user daemon-reload
```

## Enable

```bash
systemctl --user enable helix.target
```

## Start

```bash
systemctl --user start helix.target
```

## Check Status

```bash
systemctl --user status helix.target
```

## View Individual Services

```bash
systemctl --user status helix-server.service
systemctl --user status helix-postgres.service
systemctl --user status helix-nats.service
```

## View Logs

```bash
journalctl --user -u helix.target -f
journalctl --user -u helix-server.service -f
```
