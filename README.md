# TG SSH Bot

Execute commands on remote servers directly from Telegram. No servers to manage — just your bot and an SSH client.

## Features

- **AES-256-GCM encryption** — passwords encrypted at rest with unique IV per entry
- **Session persistence** — saves last SSH session per user for instant `/reconnect`
- **Configurable port** — supports any SSH port (defaults to 22)
- **Terminal-like UX** — send any message as a command, get formatted output
- **User whitelist** — restrict access to specific Telegram user IDs
- **Auto-cleanup** — password messages deleted after input

## Quick Start

### 1. Create a Telegram bot

1. Open [@BotFather](https://t.me/BotFather) in Telegram
2. Send `/newbot` and follow the prompts
3. Copy the bot token

### 2. Get your Telegram user ID

Send any message to [@userinfobot](https://t.me/userinfobot) — it replies with your numeric ID.

### 3. Set up the project

```bash
git clone <your-repo-url>
cd tg-ssh-bot
npm install
```

### 4. Configure

```bash
cp .env.example .env
```

Edit `.env`:

```env
BOT_TOKEN=123456:ABC-DEF...
ENCRYPTION_KEY=c402dcc4bed5bba11ce91136fea6da5a4332c3d3f52f444af3f87d5e96794b08
ALLOWED_USERS=123456789
```

Generate a new encryption key if you want:

```bash
node -e "console.log(require('crypto').randomBytes(32).toString('hex'))"
```

### 5. Run

```bash
npm start
```

Or with auto-reload during development:

```bash
npm run dev
```

## Usage

### Commands

| Command | Description |
|---------|-------------|
| `/start` | Show welcome message |
| `/connect` | Connect to a new SSH server (step-by-step prompt) |
| `/reconnect` | Reconnect to last saved server |
| `/disconnect` | End current SSH session |
| `/status` | Show connection state |
| `/cancel` | Cancel connection setup |

### Connecting

1. Send `/connect`
2. Enter the **host** (e.g. `192.168.1.100` or `myserver.com`)
3. Enter the **port** (leave empty for 22)
4. Enter the **username**
5. Enter the **password** (auto-deleted for security)

Once connected, **any message you send is executed as a command** on the remote server.

### Example Session

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
        Type /disconnect to end session.

You:    uname -a
Bot:    🖥 root@myserver.com:2222

        $ uname -a

        Linux myserver 5.15.0 #1 SMP x86_64 GNU/Linux

        156ms

You:    df -h
Bot:    🖥 root@myserver.com:2222

        $ df -h

        Filesystem      Size  Used Avail Use% Mounted on
        /dev/sda1        50G   12G   36G  25% /

        89ms
```

## Project Structure

```
tg-ssh-bot/
├─ .env.example          # Config template
├─ .env                  # Your config (git-ignored)
├─ package.json
└─ src/
   ├─ index.js           # Bot logic & commands
   ├─ db.js              # SQLite session storage
   ├─ ssh.js             # SSH client wrapper
   └─ utils/
      ├─ encryption.js   # AES-256-GCM encrypt/decrypt
      └── output.js      # ANSI strip, formatting, chunking
```

## Security

- **Passwords encrypted** with AES-256-GCM before storing in SQLite
- **Unique IV** per encryption — same password produces different ciphertext
- **Master key** from env var — never committed to git
- **Password messages** auto-deleted after bot reads them
- **User whitelist** — only approved Telegram IDs can use the bot
- **DM only** — ignores group messages

## Limitations

- One SSH session per user at a time
- No interactive programs (vim, top, htop) — command-exec only via `exec`
- Command timeout at 30 seconds
- 30-second inactivity cleanup for dead sessions

## License

MIT
