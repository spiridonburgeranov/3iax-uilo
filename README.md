[English](/README.md) · [Русский](/README.ru_RU.md)

<p align="center">
  <img src="./media/3iax-uilo.png" alt="3iax-uilo" width="420">
</p>

<h1 align="center">3iax-uilo</h1>

<p align="center">
  <strong>A fork of the latest <a href="https://github.com/MHSanaei/3x-ui">3x-ui</a> with AmneziaWG&nbsp;v2 baked in.</strong><br>
  Same panel you know — plus native AWG runtime, smarter traffic, and a few opinionated UI tweaks.
</p>

<p align="center">
  <a href="https://github.com/spiridonburgeranov/3iax-uilo"><img src="https://img.shields.io/github/v/release/spiridonburgeranov/3iax-uilo?style=flat-square" alt="Release"></a>
  <a href="https://github.com/spiridonburgeranov/3iax-uilo/actions"><img src="https://img.shields.io/github/actions/workflow/status/spiridonburgeranov/3iax-uilo/release.yml?style=flat-square" alt="Build"></a>
  <a href="https://github.com/MHSanaei/3x-ui"><img src="https://img.shields.io/badge/upstream-3x--ui-blue?style=flat-square" alt="Upstream"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20v3-blue?style=flat-square" alt="License"></a>
</p>

---

> [!CAUTION]
> **Vibe-coded disclaimer**
>
> This repository is **100% vibe-coded**. No human sat down and hand-wrote the diff line by line — features, fixes, refactors, and README included were produced with AI assistance (Cursor / agents) from high-level prompts.
>
> Treat it as an experiment, not a audited product. **No warranty, no production promises, no “enterprise-grade” anything.** If something breaks at 3 a.m., that’s between you and your stack trace.
>
> You are responsible for how you deploy and use it. Personal / lab use only.

> [!IMPORTANT]
> Forked from **[MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)** (currently based on **v3.4.2**). Core architecture, Xray integration, and most of the panel still come from upstream. Documentation for the base product lives in the [3x-ui Wiki](https://github.com/MHSanaei/3x-ui/wiki).

---

## What is this?

**3iax-uilo** is a web control panel for **[Xray-core](https://github.com/XTLS/Xray-core)** — the same role as upstream 3x-ui: manage inbounds, clients, routing, subscriptions, nodes, and traffic from a browser.

This fork exists because we wanted **AmneziaWG v2** to feel first-class inside the same UI you already use for VLESS / VMess / Trojan / Shadowsocks — not as a side script or a separate admin tool.

| | Upstream 3x-ui | 3iax-uilo (this fork) |
| --- | --- | --- |
| Base | Full Xray panel, multi-node, subscriptions, Telegram bot, … | Same codebase lineage |
| WireGuard in Xray | Yes | Yes |
| **AmneziaWG v2 (host `awg`)** | No | **Yes** — runtime peers, scan/import, dedicated AWG page |
| Multi-inbound client sessions | Email-level “online” only | **Per-inbound session attribution** (which tag the client actually uses) |
| AWG traffic | — | Per-inbound speed / counters in the panel |

---

## Fork highlights

### AmneziaWG v2

- Inbound protocol **`amneziawg`** managed like any other inbound (create, enable, clients, QR / config export).
- Host integration via **`awg` / `awg-quick`** — apply config, bring interfaces up/down, read runtime peer stats.
- **Startup scan** — discover existing AWG interfaces on the host and import them into the panel.
- **AWG dashboard** — runtime status, interface summary, without dumping raw peer tables on the home screen.
- Client configs use a sensible **server endpoint** (public IP when possible, not a placeholder hostname).

### Traffic & “who is online where”

- AWG polls feed **per-inbound** traffic into the same accounting pipeline as Xray.
- Clients attached to **several inbounds** show **one** active session: the panel tracks `email → inbound tag` each poll window instead of lighting up every attachment.

### Everything else (from upstream)

- Protocols: VLESS, VMess, Trojan, Shadowsocks, WireGuard, Hysteria2, MTProto, HTTP, SOCKS, TUN, …
- Transports: TCP, mKCP, WebSocket, gRPC, HTTPUpgrade, XHTTP + TLS / REALITY
- Per-client quotas, expiry, IP limits, subscriptions (raw / JSON / Clash)
- Multi-node master ↔ sub-node sync
- SQLite or PostgreSQL
- React 19 + Ant Design 6 frontend, 13 UI languages, dark / light themes
- REST API + in-panel OpenAPI docs, Telegram bot, Fail2ban IP limits

---

## Quick start

```bash
bash <(curl -Ls https://raw.githubusercontent.com/spiridonburgeranov/3iax-uilo/main/install.sh)
```

Install a specific release tag:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/spiridonburgeranov/3iax-uilo/main/install.sh) v3.4.2
```

After install, run `x-ui` for the management menu (start/stop, credentials, SSL, logs).

### AWG on the host

The installer can pull in **AmneziaWG** tooling where supported (see `install.sh`). You still need a compatible kernel / module on Linux. Standard **WireGuard clients are not enough** for AmneziaWG — use [AmneziaVPN](https://github.com/amnezia-vpn/amnezia-client) or another AWG-capable app.

### Build from source

```bash
git clone https://github.com/spiridonburgeranov/3iax-uilo.git
cd 3iax-uilo
make build    # needs Go (CGO for SQLite), Node for frontend
```

Frontend dev loop: `cd frontend && npm run dev` (proxies to the Go panel on `:2053`).

---

## Platforms

**OS:** Ubuntu, Debian, Fedora, RHEL family, Arch, Alpine, Windows, and most Linux distros upstream supports.

**Arch:** `amd64` · `386` · `arm64` · `armv7` · `armv6` · `armv5` · `s390x`

---

## Database

Same as upstream:

- **SQLite** (default) — `/etc/x-ui/x-ui.db`
- **PostgreSQL** — set `XUI_DB_TYPE=postgres` and `XUI_DB_DSN` (installer or `/etc/default/x-ui`)

```bash
x-ui migrate-db --dsn "postgres://xui:password@127.0.0.1:5432/xui?sslmode=disable"
```

---

## Docker

```bash
docker compose up -d
# PostgreSQL profile:
docker compose --profile postgres up -d
```

Fail2ban inside the image needs `NET_ADMIN` / `NET_RAW` (already in `docker-compose.yml`).

---

## Environment variables

Same surface as upstream 3x-ui — see [upstream README](https://github.com/MHSanaei/3x-ui/blob/main/README.md#environment-variables) for the full table (`XUI_DB_*`, `XUI_LOG_LEVEL`, `XUI_DEBUG`, tunnel health monitor, …).

---

## Credits & license

- **Upstream:** [3x-ui](https://github.com/MHSanaei/3x-ui) by [@MHSanaei](https://github.com/MHSanaei) and contributors — **GPL-3.0**
- **AmneziaWG:** [amnezia-vpn](https://github.com/amnezia-vpn) ecosystem
- **This fork:** maintenance and vibe-coded patches on top — still **GPL-3.0**

If upstream helped you before, consider starring **[MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)** too.

---

<p align="center">
  <sub>Made with prompts, caffeine, and questionable confidence.</sub>
</p>
