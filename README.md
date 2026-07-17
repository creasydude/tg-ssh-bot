# TG SSH Bot

Execute commands on remote servers directly from Telegram. Single static binary — no runtime dependencies.

## Features

- **Single binary** — one file, no node_modules, no runtime deps
- **AES-256-GCM encryption** — passwords encrypted at rest with unique nonce
- **Session persistence** — saves last SSH session for instant `/reconnect`
- **Configurable port** — supports any SSH port
- **Terminal-like UX** — formatted output with server info and execution time
- **User whitelist** — restrict access to specific Telegram IDs
- **SQLite** — lightweight embedded database

## Quick Start

### Option 1: Download the binary

Go to [Releases](../../releases) and download the latest `tg-ssh-bot` binary.

```bash
chmod +x tg-ssh-bot
cp .env.example .env
# Edit .env with your credentials
./tg-ssh-bot
```

### Option 2: Build from source

```bash
git clone <your-repo-url>
cd tg-ssh-bot
go build -ldflags="-s -w" -o tg-ssh-bot .
```

### Configuration

```bash
cp .env.example .env
```

Edit `.env`:

```env
BOT_TOKEN=123456:ABC-DEF...
ENCRYPTION_KEY=ee40de55b3eb1faceed105109453742e308d1a854ea982520f7ab8a60ea84846
ALLOWED_USERS=123456789
```

Generate a new encryption key:

```bash
openssl rand -hex 32
```

### Run

```bash
./tg-ssh-bot
```

## Commands

| Command | Description |
|---------|-------------|
| `/start` | Show welcome message |
| `/connect` | Connect to a new SSH server |
| `/reconnect` | Reconnect to last saved server |
| `/disconnect` | End current SSH session |
| `/status` | Show connection state |
| `/cancel` | Cancel connection setup |

## Example Session

```
You:    /connect
Bot:    Enter SSH host:
You:    myserver.com
Bot:    Port (default 22):
You:    2222
Bot:    Username:
You:    root
Bot:    Password:
You:    ****  ← (auto-deleted)
Bot:    ✅ Connected
        root@myserver.com:2222

        Send any message to execute commands.

You:    uname -a
Bot:    🖥 root@myserver.com:2222

        $ uname -a

        Linux myserver 5.15.0 #1 SMP x86_64 GNU/Linux

        156ms
```

## Project Structure

```
tg-ssh-bot/
├─ main.go              # Entry point, config
├─ bot.go               # Telegram bot, commands, conversation flow
├─ ssh.go               # SSH client
├─ db.go                # SQLite database
├─ crypto.go            # AES-256-GCM encryption
├─ output.go            # ANSI stripping, formatting
├─ .env                 # Config (git-ignored)
└─ .github/workflows/
   └─ build.yml         # GitHub Actions — builds static binary
```

## GitHub Actions

Push a tag to trigger a build and release:

```bash
git tag v1.0.0
git push --tags
```

The workflow builds a static Linux AMD64 binary and attaches it to the GitHub release.

## Security

- **AES-256-GCM** encryption with random nonce per entry
- **Master key** from env var — never committed
- **Password messages** auto-deleted after input
- **User whitelist** — only approved Telegram IDs
- **DM only** — ignores group messages

## License

MIT
