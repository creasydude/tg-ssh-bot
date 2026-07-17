// Strip ANSI escape codes
export function stripAnsi(str) {
  return str
    .replace(/\x1B\[[0-9;]*[a-zA-Z]/g, '')
    .replace(/\x1B\][^\x07]*\x07/g, '')
    .replace(/\x1B[()][AB012]/g, '');
}

// Clean trailing whitespace from each line, trim overall
export function cleanOutput(raw) {
  return stripAnsi(raw)
    .split('\n')
    .map((l) => l.replace(/\r/g, '').replace(/\s+$/, ''))
    .join('\n')
    .trim();
}

// Split text into chunks under Telegram's 4096 char limit
export function chunkMessage(text, limit = 4000) {
  if (text.length <= limit) return [text];

  const chunks = [];
  let remaining = text;

  while (remaining.length > 0) {
    if (remaining.length <= limit) {
      chunks.push(remaining);
      break;
    }

    let cutAt = remaining.lastIndexOf('\n', limit);
    if (cutAt <= 0) cutAt = limit;

    chunks.push(remaining.substring(0, cutAt));
    remaining = remaining.substring(cutAt).replace(/^\n/, '');
  }

  return chunks;
}

function escHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// Format a successful command result
export function formatSuccess(cmd, output, serverInfo, execTime) {
  const timeStr = execTime !== null ? `${execTime}ms` : '';
  const header = serverInfo
    ? `<b>\u{1F5A5} ${escHtml(serverInfo)}</b>`
    : '';

  const cmdLine = `<code>$ ${escHtml(cmd)}</code>`;

  if (!output) {
    const parts = [header, cmdLine, '<i>(no output)</i>'];
    if (timeStr) parts.push(`<i>${timeStr}</i>`);
    return parts.filter(Boolean).join('\n\n');
  }

  const parts = [header, cmdLine, `<pre>${escHtml(output)}</pre>`];
  if (timeStr) parts.push(`<i>${timeStr}</i>`);
  return parts.join('\n\n');
}

// Format an error result
export function formatError(cmd, error, serverInfo) {
  const header = serverInfo
    ? `<b>\u{1F5A5} ${escHtml(serverInfo)}</b>`
    : '';

  const parts = [
    header,
    `<code>$ ${escHtml(cmd)}</code>`,
    `<i>\u274C ${escHtml(error)}</i>`,
  ];
  return parts.filter(Boolean).join('\n\n');
}

// Format connection status messages
export function formatConnected(host, port, username) {
  return [
    `\u2705 <b>Connected</b>`,
    `<code>${escHtml(username)}@${escHtml(host)}:${port}</code>`,
    '',
    `Send any message to execute commands.`,
    `Type /disconnect to end session.`,
  ].join('\n');
}

export function formatConnecting(host, port, username) {
  return `\u23F3 Connecting to <code>${escHtml(username)}@${escHtml(host)}:${port}</code>...`;
}

export function formatConnectionError(error) {
  return `\u274C Connection failed: <i>${escHtml(error)}</i>`;
}
