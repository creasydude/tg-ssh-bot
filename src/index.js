import 'dotenv/config';
import { Bot } from 'grammy';
import { parseKey, encrypt, decrypt } from './utils/encryption.js';
import {
  cleanOutput,
  chunkMessage,
  formatSuccess,
  formatError,
  formatConnected,
  formatConnecting,
  formatConnectionError,
} from './utils/output.js';
import { initDb, saveSession, getSession, deleteSession } from './db.js';
import { SshSession } from './ssh.js';

// --- Config ---
const BOT_TOKEN = process.env.BOT_TOKEN;
const ENCRYPTION_KEY_HEX = process.env.ENCRYPTION_KEY;
const ALLOWED_USERS = (process.env.ALLOWED_USERS || '')
  .split(',')
  .map((s) => parseInt(s.trim(), 10))
  .filter(Boolean);

if (!BOT_TOKEN) throw new Error('BOT_TOKEN is required');
if (!ENCRYPTION_KEY_HEX) throw new Error('ENCRYPTION_KEY is required');

const encryptionKey = parseKey(ENCRYPTION_KEY_HEX);
const db = initDb();
const bot = new Bot(BOT_TOKEN);

// --- State ---
const sshSessions = new Map();   // userId -> SshSession
const conversations = new Map(); // userId -> { step, data }

const STEPS = { HOST: 'host', PORT: 'port', USER: 'user', PASS: 'pass' };

// --- Middleware: auth + DM only ---
bot.use(async (ctx, next) => {
  if (!ctx.from) return;
  if (ctx.chat?.type !== 'private') return; // only DMs
  if (ALLOWED_USERS.length > 0 && !ALLOWED_USERS.includes(ctx.from.id)) {
    return ctx.reply('Access denied.');
  }
  return next();
});

// --- Commands ---

bot.command('start', (ctx) =>
  ctx.reply(
    [
      '*SSH Bot*',
      '',
      'Execute commands on remote servers via Telegram.',
      '',
      '*Commands:*',
      '/connect \\- Connect to an SSH server',
      '/reconnect \\- Reconnect to last saved server',
      '/disconnect \\- Disconnect current session',
      '/status \\- Show connection status',
      '/cancel \\- Cancel ongoing connection setup',
      '/help \\- Show this message',
      '',
      '_All passwords are encrypted at rest._',
    ].join('\n'),
    { parse_mode: 'Markdown' }
  )
);

bot.command('help', (ctx) => bot.handlers.start.handler(ctx));

bot.command('status', (ctx) => {
  const session = sshSessions.get(ctx.from.id);
  if (session?.connected) {
    const saved = getSession(ctx.from.id);
    if (saved) {
      return ctx.reply(
        `\u{1F7E2} <b>Connected</b>\n<code>${saved.username}@${saved.host}:${saved.port}</code>`,
        { parse_mode: 'HTML' }
      );
    }
    return ctx.reply('\u{1F7E2} <b>Connected</b> (session info unavailable)', { parse_mode: 'HTML' });
  }
  const saved = getSession(ctx.from.id);
  if (saved) {
    return ctx.reply(
      `\u{1F534} <b>Disconnected</b>\nLast: <code>${saved.username}@${saved.host}:${saved.port}</code>\nUse /reconnect`,
      { parse_mode: 'HTML' }
    );
  }
  return ctx.reply('\u{1F534} <b>No sessions</b>\nUse /connect to start', { parse_mode: 'HTML' });
});

bot.command('cancel', (ctx) => {
  if (conversations.has(ctx.from.id)) {
    conversations.delete(ctx.from.id);
    return ctx.reply('Connection setup cancelled.');
  }
  return ctx.reply('Nothing to cancel.');
});

bot.command('disconnect', (ctx) => {
  const session = sshSessions.get(ctx.from.id);
  if (session) {
    session.disconnect();
    sshSessions.delete(ctx.from.id);
    return ctx.reply('\u{1F50C} Disconnected.');
  }
  return ctx.reply('No active connection.');
});

bot.command('reconnect', async (ctx) => {
  const saved = getSession(ctx.from.id);
  if (!saved) {
    return ctx.reply('No saved session. Use /connect first.');
  }

  // Disconnect existing
  const existing = sshSessions.get(ctx.from.id);
  if (existing) existing.disconnect();

  const password = decrypt(saved.password, encryptionKey);
  const session = new SshSession();

  const msg = await ctx.reply(formatConnecting(saved.host, saved.port, saved.username), {
    parse_mode: 'HTML',
  });

  try {
    await session.connect({
      host: saved.host,
      port: saved.port,
      username: saved.username,
      password,
    });
    sshSessions.set(ctx.from.id, session);
    await ctx.api.editMessageText(ctx.chat.id, msg.message_id,
      formatConnected(saved.host, saved.port, saved.username),
      { parse_mode: 'HTML' }
    );
  } catch (err) {
    await ctx.api.editMessageText(ctx.chat.id, msg.message_id,
      formatConnectionError(err.message),
      { parse_mode: 'HTML' }
    );
  }
});

bot.command('connect', (ctx) => {
  // Disconnect existing
  const existing = sshSessions.get(ctx.from.id);
  if (existing) {
    existing.disconnect();
    sshSessions.delete(ctx.from.id);
  }

  conversations.set(ctx.from.id, { step: STEPS.HOST, data: {} });
  ctx.reply('Enter SSH host:');
});

// --- Conversation handler (state machine) ---
async function handleConversation(ctx) {
  const conv = conversations.get(ctx.from.id);
  if (!conv) return false;

  const text = ctx.message?.text;
  if (!text) return false;

  switch (conv.step) {
    case STEPS.HOST: {
      const host = text.trim();
      if (!host || host.length > 255) {
        return ctx.reply('Invalid host. Try again:');
      }
      conv.data.host = host;
      conv.step = STEPS.PORT;
      return ctx.reply(`Port (default 22):`);
    }
    case STEPS.PORT: {
      const port = text.trim() === '' ? 22 : parseInt(text.trim(), 10);
      if (isNaN(port) || port < 1 || port > 65535) {
        return ctx.reply('Invalid port. Enter a number 1-65535:');
      }
      conv.data.port = port;
      conv.step = STEPS.USER;
      return ctx.reply('Username:');
    }
    case STEPS.USER: {
      const username = text.trim();
      if (!username) {
        return ctx.reply('Username cannot be empty. Try again:');
      }
      conv.data.username = username;
      conv.step = STEPS.PASS;
      return ctx.reply('Password:');
    }
    case STEPS.PASS: {
      const password = text;
      conv.data.password = password;

      // Delete the password message for security
      try {
        await ctx.deleteMessage();
      } catch {
        // Can't delete if bot doesn't have delete rights
      }

      const { host, port, username } = conv.data;
      conversations.delete(ctx.from.id);

      // Attempt connection
      const session = new SshSession();
      const msg = await ctx.reply(formatConnecting(host, port, username), {
        parse_mode: 'HTML',
      });

      try {
        await session.connect({ host, port, username, password });
        sshSessions.set(ctx.from.id, session);

        // Save session (encrypted)
        const encryptedPassword = encrypt(password, encryptionKey);
        saveSession(ctx.from.id, { host, port, username, encryptedPassword });

        await ctx.api.editMessageText(
          ctx.chat.id,
          msg.message_id,
          formatConnected(host, port, username),
          { parse_mode: 'HTML' }
        );
      } catch (err) {
        await ctx.api.editMessageText(
          ctx.chat.id,
          msg.message_id,
          formatConnectionError(err.message),
          { parse_mode: 'HTML' }
        );
      }
      return true;
    }
  }
  return false;
}

// --- Message handler: commands or SSH execution ---
bot.on('message:text', async (ctx) => {
  const text = ctx.message.text;

  // Check for conversation state first
  if (conversations.has(ctx.from.id)) {
    return handleConversation(ctx);
  }

  // Check for active SSH session
  const session = sshSessions.get(ctx.from.id);
  if (!session?.connected) {
    return; // ignore non-command messages when not connected
  }

  // Build server info string from saved session
  const saved = getSession(ctx.from.id);
  const serverInfo = saved
    ? `${saved.username}@${saved.host}:${saved.port}`
    : null;

  // Send typing indicator
  await ctx.replyWithChatAction('typing');

  const startTime = Date.now();

  try {
    const raw = await session.exec(text);
    const execTime = Date.now() - startTime;
    const output = cleanOutput(raw);
    const formatted = formatSuccess(text, output, serverInfo, execTime);
    const chunks = chunkMessage(formatted);

    for (const chunk of chunks) {
      await ctx.reply(chunk, { parse_mode: 'HTML' });
    }
  } catch (err) {
    // Connection lost — clean up
    sshSessions.delete(ctx.from.id);
    const formatted = formatError(text, `${err.message}\nSession lost. Use /reconnect.`, serverInfo);
    await ctx.reply(formatted, { parse_mode: 'HTML' });
  }
});

// --- Handle disconnected sessions ---
function setupSessionWatchers() {
  // Periodically check for dead sessions
  setInterval(() => {
    for (const [userId, session] of sshSessions) {
      if (!session.connected) {
        sshSessions.delete(userId);
      }
    }
  }, 30000);
}

// --- Start ---
setupSessionWatchers();

console.log('SSH Bot starting...');
bot.start({
  onStart: () => console.log('Bot is running'),
  drop_pending_updates: true,
});
