# Age Encrypted Notebook (aen)
*Disclaimer: This project has the sole purpose of getting me into Go development. I just want to play around with Go a bit.*

`aen` uses Age ([github.com/FiloSottile/age](https://github.com/FiloSottile/age)) to encrypt text snippets ("notes") and bolt ([github.com/etcd-io/bbolt](https://github.com/etcd-io/bbolt)) to store them in a k/v database. This can be useful for e.g. transporting encrypted data to airgapped systems without the hassle of shared keys (well, after an initial setup :) ), the DB then resides on a removable media. Keep in mind that creating the note with an external editor (`create` command) or editing a note in a later version of aen requires to write the note unencrypted to a file. While the file is deleted afterwards, it can be recovered if you not choose to overwrite it with random data afterwards (`-S/--shred`).

## Usage
```
Age Encrypted Notebook (devel)

* DB and keyfile paths can also be given via environment variables AENDB and AENKEY.
** The default editor can be changed through setting the environment variable AENEDITOR.

Usage:

aen init (in)          Initializes the private key and the database if not already given
                       and adds the own public key to the database
  -o, --output         - Path to DB *
  -k, --key            - Path to age keyfile *

aen list (ls)          Lists the slugs of available notes sorted by their timestamp
  -d, --db             - Path to DB *

aen create (cr)        Creates a new note with an editor using the first line of the created
                       note as title
                       By default the command calls 'codium -w' **
  -d, --db             - Path to DB *
  -S, --shred          - Overwrites temporary file with random data

aen edit (ed)          Edits a note given by slug or id
                       By default the command calls 'codium -w' **
  -d, --db             - Path to DB *
  -k, --key            - Path to age keyfile *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get
  -S, --shred          - Overwrites temporary file with random data

aen write (wr)         Writes a new note
  -d, --db             - Path to DB *
  -t, --title          - Title of the note
  -m, --message        - Message of the note

aen get (g)            Get and decrypt a note by its slug or id
  -d, --db             - Path to DB *
  -k, --key            - Path to age keyfile *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get
  -r, --raw            - Only print note content without any metadata

aen remove (rm)        Removes note by its slug or id from the database
                       NOTE: While the note is not retrievable through aen anymore,
                       the data reside in the database file until its overwritten by a new note.
  -d, --db             - Path to DB *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get

aen recipients (re)   Lists all recipients and their aliases
  -d, --db            - Path to DB *
  -r, --remove        - Remove recipient identified by its alias
```
First, if not already available, key and database must be generated using `aen init`. This generates a key as well as the database and adds the public key as recipient to the DB. Then other commands can be used to handle encrypted notes in the database. The database path as well as the key path and the editor command can also be set via environment variables:

```
# Defaults
AENDB=""
AENKEY=""
AENEDITOR="codium -w"
```

Be aware that the first line of the note created with `aen create` will be used as a title. Every character matching `[^a-zA-Z0-9 !\"§$%&/()=]+` will be removed from that.

## Example
The following example snippet shows the initialization of the database as well as adding, viewing and deleting a note.

```
❯ export AENDB=/tmp/test.db
❯ export AENKEY=/tmp/aen_1
❯ aen init
Written key to /tmp/aen_1.
Public key: age1xphzytv7l6jta9a5cczes0agg5aq37ewrcpc54y5mehnjsqlw48qr0wyc7
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