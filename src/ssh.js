import { Client } from 'ssh2';

export class SshSession {
  constructor() {
    this.conn = null;
    this.stream = null;
    this.connected = false;
  }

  connect({ host, port, username, password }) {
    return new Promise((resolve, reject) => {
      this.conn = new Client();

      const timeout = setTimeout(() => {
        this.conn.end();
        reject(new Error('Connection timed out (10s)'));
      }, 10000);

      this.conn.on('ready', () => {
        clearTimeout(timeout);
        this.conn.shell({ term: 'xterm-256color', cols: 120, rows: 40 }, (err, stream) => {
          if (err) {
            this.conn.end();
            return reject(err);
          }
          this.stream = stream;
          this.connected = true;

          // Dismiss the initial shell prompt by reading and discarding it
          let initBuffer = '';
          const initHandler = (data) => {
            initBuffer += data.toString();
            // Once we see a prompt-like pattern or after a short delay, stop buffering
          };
          stream.on('data', initHandler);

          setTimeout(() => {
            stream.removeListener('data', initHandler);
            resolve();
          }, 500);
        });
      });

      this.conn.on('error', (err) => {
        clearTimeout(timeout);
        this.connected = false;
        reject(err);
      });

      this.conn.on('close', () => {
        this.connected = false;
        this.stream = null;
      });

      this.conn.connect({ host, port, username, password });
    });
  }

  exec(command) {
    return new Promise((resolve, reject) => {
      if (!this.connected || !this.stream) {
        return reject(new Error('Not connected'));
      }

      let output = '';
      let closed = false;

      // Use exec for cleaner output (no prompt artifacts)
      this.conn.exec(command, (err, stream) => {
        if (err) return reject(err);

        stream.on('data', (data) => {
          output += data.toString();
        });

        stream.stderr.on('data', (data) => {
          output += data.toString();
        });

        stream.on('close', () => {
          if (!closed) {
            closed = true;
            resolve(output);
          }
        });

        stream.on('error', (err) => {
          if (!closed) {
            closed = true;
            reject(err);
          }
        });

        // Safety timeout for hanging commands
        setTimeout(() => {
          if (!closed) {
            closed = true;
            stream.close();
            resolve(output + '\n\n[timed out after 30s]');
          }
        }, 30000);
      });
    });
  }

  disconnect() {
    if (this.stream) {
      this.stream.close();
    }
    if (this.conn) {
      this.conn.end();
    }
    this.connected = false;
    this.stream = null;
    this.conn = null;
  }
}
