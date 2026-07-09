[English](/README.md) · [Русский](/README.ru_RU.md)

<p align="center">
  <img src="./media/3iax-uilo.png" alt="3iax-uilo" width="420">
</p>

<h1 align="center">3iax-uilo</h1>

<p align="center">
  <strong>Форк актуальной версии <a href="https://github.com/MHSanaei/3x-ui">3x-ui</a> со встроенным AmneziaWG&nbsp;v2.</strong><br>
  Та же панель, что вы знаете — плюс нативный AWG, умнее трафик и пара UI-правок по вкусу.
</p>

<p align="center">
  <a href="https://github.com/spiridonburgeranov/3iax-uilo"><img src="https://img.shields.io/github/v/release/spiridonburgeranov/3iax-uilo?style=flat-square" alt="Release"></a>
  <a href="https://github.com/spiridonburgeranov/3iax-uilo/actions"><img src="https://img.shields.io/github/actions/workflow/status/spiridonburgeranov/3iax-uilo/release.yml?style=flat-square" alt="Build"></a>
  <a href="https://github.com/MHSanaei/3x-ui"><img src="https://img.shields.io/badge/upstream-3x--ui-blue?style=flat-square" alt="Upstream"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20v3-blue?style=flat-square" alt="License"></a>
</p>

---

> [!CAUTION]
> **Дисклеймер: всё завайбкожено**
>
> Этот репозиторий **на 100% vibe-coded**. Человек не сидел и не писал код построчно — фичи, фиксы, рефакторинг и даже этот README собраны с помощью ИИ (Cursor / агенты) из описаний «сделай вот так».
>
> Относитесь к проекту как к эксперименту, а не к аудированному продукту. **Никаких гарантий, никакого «enterprise», никакой магии.** Если в 3 ночи что-то упало — разбирайтесь со stack trace сами.
>
> За способ развёртывания и использования отвечаете вы. Только личное / лабораторное применение.

> [!IMPORTANT]
> Форк **[MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)** (сейчас ориентир — **v3.4.2**). Архитектура, интеграция с Xray и большая часть панели — от upstream. Базовая документация: [вики 3x-ui](https://github.com/MHSanaei/3x-ui/wiki).

---

## Что это?

**3iax-uilo** — веб-панель для **[Xray-core](https://github.com/XTLS/Xray-core)**. Те же задачи, что у оригинального 3x-ui: инбаунды, клиенты, маршрутизация, подписки, ноды, статистика — из браузера.

Форк появился потому, что **AmneziaWG v2** хотелось видеть в том же интерфейсе, что VLESS / VMess / Trojan / Shadowsocks — без отдельных скриптов и «админки сбоку».

| | Оригинал 3x-ui | 3iax-uilo (этот форк) |
| --- | --- | --- |
| База | Полная Xray-панель, мульти-ноды, подписки, Telegram-бот, … | Та же кодовая база |
| WireGuard в Xray | Да | Да |
| **AmneziaWG v2 (хост `awg`)** | Нет | **Да** — runtime, скан/импорт, страница AWG |
| Сессия при нескольких инбаундах | «Онлайн» только по email | **Привязка к конкретному inbound tag** |
| Трафик AWG | — | Скорость и счётчики **по инбаунду** |

---

## Чем отличается форк

### AmneziaWG v2

- Протокол **`amneziawg`** в панели как обычный инбаунд: создание, клиенты, QR / выдача конфига.
- Интеграция с хостом через **`awg` / `awg-quick`**: поднять интерфейс, статистика пиров, apply конфига.
- **Скан при старте** — найти существующие AWG-интерфейсы на сервере и импортировать в БД.
- **Дашборд AWG** — статус runtime и сводка без простыни peer-ов на главной.
- В клиентских конфигах **нормальный endpoint** (публичный IP, а не заглушка `awg`).

### Трафик и «кто через какой инбаунд»

- Опрос AWG отдаёт **побайтный трафик по инбаундам** в общую систему учёта.
- Клиент на **двух инбаундах** подсвечивается онлайн **только на том**, через который реально идёт сессия (`email → tag` на каждом poll-окне).

### Остальное — от upstream

- Протоколы: VLESS, VMess, Trojan, Shadowsocks, WireGuard, Hysteria2, MTProto, HTTP, SOCKS, TUN, …
- Транспорты: TCP, mKCP, WebSocket, gRPC, HTTPUpgrade, XHTTP + TLS / REALITY
- Квоты, срок действия, лимит IP, подписки (raw / JSON / Clash)
- Синхронизация master ↔ sub-node
- SQLite или PostgreSQL
- React 19 + Ant Design 6, 13 языков интерфейса, тёмная / светлая тема
- REST API + OpenAPI в панели, Telegram-бот, Fail2ban

---

## Быстрый старт

```bash
bash <(curl -Ls https://raw.githubusercontent.com/spiridonburgeranov/3iax-uilo/main/install.sh)
```

Конкретный релиз:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/spiridonburgeranov/3iax-uilo/main/install.sh) v3.4.2
```

После установки: `x-ui` — меню (старт/стоп, логин, SSL, логи).

### AWG на сервере

Установщик по возможности ставит **AmneziaWG** (см. `install.sh`). Нужны совместимые ядро / модуль на Linux. Обычные **WireGuard-клиенты не подходят** для AmneziaWG — берите [AmneziaVPN](https://github.com/amnezia-vpn/amnezia-client) или другое AWG-приложение.

### Сборка из исходников

```bash
git clone https://github.com/spiridonburgeranov/3iax-uilo.git
cd 3iax-uilo
make build    # Go (CGO для SQLite) + Node для фронта
```

Разработка UI: `cd frontend && npm run dev` (прокси на Go-панель `:2053`).

---

## Платформы

**ОС:** Ubuntu, Debian, Fedora, RHEL-семейство, Arch, Alpine, Windows и прочее, что тянет upstream.

**Архитектуры:** `amd64` · `386` · `arm64` · `armv7` · `armv6` · `armv5` · `s390x`

---

## База данных

Как в upstream:

- **SQLite** (по умолчанию) — `/etc/x-ui/x-ui.db`
- **PostgreSQL** — `XUI_DB_TYPE=postgres` и `XUI_DB_DSN`

```bash
x-ui migrate-db --dsn "postgres://xui:password@127.0.0.1:5432/xui?sslmode=disable"
```

---

## Docker

```bash
docker compose up -d
# с PostgreSQL:
docker compose --profile postgres up -d
```

Для Fail2ban в контейнере нужны `NET_ADMIN` / `NET_RAW` (уже в `docker-compose.yml`).

---

## Переменные окружения

Тот же набор, что у 3x-ui — полная таблица в [README upstream](https://github.com/MHSanaei/3x-ui/blob/main/README.md#environment-variables) (`XUI_DB_*`, `XUI_LOG_LEVEL`, `XUI_DEBUG`, health monitor туннеля, …).

---

## Благодарности и лицензия

- **Upstream:** [3x-ui](https://github.com/MHSanaei/3x-ui), [@MHSanaei](https://github.com/MHSanaei) и контрибьюторы — **GPL-3.0**
- **AmneziaWG:** экосистема [amnezia-vpn](https://github.com/amnezia-vpn)
- **Этот форк:** патчи сверху — тоже **GPL-3.0**

Если оригинал вам зашёл — поставьте звезду **[MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)**.

---

<p align="center">
  <sub>Сделано промптами, кофеином и сомнительной уверенностью.</sub>
</p>
