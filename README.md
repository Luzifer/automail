[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/automail)](https://goreportcard.com/report/github.com/Luzifer/automail)
![](https://badges.fyi/github/license/Luzifer/automail)
![](https://badges.fyi/github/downloads/Luzifer/automail)
![](https://badges.fyi/github/latest-release/Luzifer/automail)
![](https://knut.in/project-status/automail)

# Luzifer / automail

`automail` is an utility to periodically fetch mails from an IMAP mailbox and execute commands based on matched headers.

One of my personal use-cases for this is to automatically parse payment receipts received from Twitch and enter the corresponding transactions into my accounting software.

In the end this software provides you with a possibility to match any mail you receive by their headers and execute a script which is able to act on those mails. The script is provided with a JSON representation of the mail on `stdin` and can yield commands (for example "mark as read", "move to mailbox", ...) to `stdout` which then will be executed on the mail.

## Storage types

- **Local File**
  - `--storage-type=file`
  - `--storage-dsn=path/to/file.yaml`
- **Redis**
  - `--storage-type=redis`
  - `--storage-dsn=redis://<user>:<password>@<host>:<port>/<db_number>`
  - `REDIS_KEY_PREFIX=myprefix` (default: `io.luzifer.automail`)
