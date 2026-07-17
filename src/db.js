import Database from 'better-sqlite3';

let db;

export function initDb(path = 'bot.db') {
  db = new Database(path);
  db.pragma('journal_mode = WAL');
  db.exec(`
    CREATE TABLE IF NOT EXISTS sessions (
      user_id    INTEGER PRIMARY KEY,
      host       TEXT    NOT NULL,
      port       INTEGER NOT NULL DEFAULT 22,
      username   TEXT    NOT NULL,
      password   TEXT    NOT NULL,
      last_used  TEXT    DEFAULT (datetime('now'))
    )
  `);
  return db;
}

export function getDb() {
  return db;
}

export function saveSession(userId, { host, port, username, encryptedPassword }) {
  const stmt = db.prepare(`
    INSERT INTO sessions (user_id, host, port, username, password, last_used)
    VALUES (?, ?, ?, ?, ?, datetime('now'))
    ON CONFLICT(user_id) DO UPDATE SET
      host      = excluded.host,
      port      = excluded.port,
      username  = excluded.username,
      password  = excluded.password,
      last_used = datetime('now')
  `);
  stmt.run(userId, host, port, username, encryptedPassword);
}

export function getSession(userId) {
  return db.prepare('SELECT * FROM sessions WHERE user_id = ?').get(userId);
}

export function deleteSession(userId) {
  db.prepare('DELETE FROM sessions WHERE user_id = ?').run(userId);
}
