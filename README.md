# Age Encrypted Notebook (aen)
*Disclaimer: This project has the sole purpose of getting me into Go development. I just want to play around with Go a bit.*

`aen` uses Age ([github.com/FiloSottile/age](https://github.com/FiloSottile/age)) to encrypt text snippets ("notes") and bolt ([github.com/etcd-io/bbolt](https://github.com/etcd-io/bbolt)) to store them in a k/v database. This can be useful for e.g. transporting encrypted data to airgapped systems without the hassle of shared keys, the DB then resides on a removable media. Obviously the only protects the notes if the removable media is at rest. On the encrypting/decrypting systems itself, the note will be availalbe unencrypted at several places (e.g. memory or in terms of note creation the `/tmp` directory).

## Usage
```
Age Encrypted Notebook v0.0.2-1-g2f9fe20

* DB and keyfile paths can also be given via evironment variables AENDB and AENKEY.
** The default editor can be changed setting the environment variable AENEDITOR.

Usage:

aen init          Initializes the private key and the database if not already given and adds the own public key to the database
  -o, --output    - Path to DB *
  -k, --key       - Path to age keyfile *

aen list          Lists the slugs of available notes sorted by their timestamp
  -d, --db        - Path to DB *

aen create        Creates a new note with an editor - by default the command calls 'codium -w' **
  -d, --db        - Path to DB *

aen write         Writes a new note
  -d, --db        - Path to DB *
  -t, --title     - Title of the note
  -m, --message   - Message of the note

aen get           Get and decrypt a note by its slug or id
  -d, --db        - Path to DB *
  -k, --key       - Path to age keyfile *
  -s, --slug      - Slug of note to get
  -i, --id        - ID of note to get

aen del           Delete note by its slug or id
  -d, --db        - Path to DB *
  -s, --slug      - Slug of note to get
  -i, --id        - ID of note to get
```
First, if not already available, key and database must be generated using `aen init`. This generates a key as well as the database and adds the public key as recipient to the DB. Then other commands can be used to handle encrypted notes in the database. The database path as well as the key path and the editor command can also be set via environment variables:

```
# Defaults
AENDB=""
AENKEY=""
AENEDITOR="codium -w"
```

## Example
The following example snippet shows the initialization of the database as well as adding, viewing and deleting a note.

```
❯ export AENDB=/tmp/test.db
❯ export AENKEY=/tmp/aen_1
❯ aen init
Written key to /tmp/aen_1.
Public key: age1xphzytv7l6jta9a5cczes0agg5aq37ewrcpc54y5mehnjsqlw48qr0wyc7
❯ aen write --title "Hello World --message "Hello from age1xphzytv7l6jta9a5cczes0agg5aq37ewrcpc54y5mehnjsqlw48qr0wyc7!"
Error writing note: title and message must be given.
❯ aen write --title "Hello World\!" --message "Hello from age1xphzytv7l6jta9a5cczes0agg5aq37ewrcpc54y5mehnjsqlw48qr0wyc7\!"
Successfully written note hello-world.
❯ aen list
| ID    | Title                     | Creation time             | Slug                      |
| 1     | Hello World!              | 2022-02-14 14:03:53       | hello-world               |
❯ aen get -i 1
Title: Hello World! (a0b6ae04-a8ae-40ca-ad6c-2debae34cd84)
Created: 2022-02-14 14:03:53
Content:
Hello from age1xphzytv7l6jta9a5cczes0agg5aq37ewrcpc54y5mehnjsqlw48qr0wyc7!
❯ aen del -i 1
```