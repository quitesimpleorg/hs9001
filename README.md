# hs9001
hs90001 (history search 9001) is an easy, quite simple bash history enhancement. It simply writes all
your bash commands into an sqlite database. You can then search this database.


## Setup
```
go build
#move hs9001 to a PATH location
# Initialize database
hs9001 init
````

Add this to .bashrc

```
if [ -n "$PS1" ] ; then
    PROMPT_COMMAND='hs9001 -ret $? add "$(history 1)"'
fi
```
By default, every system user gets his own database. You can override this by overriding the environment variable
for all users that should write to your unified database.

```
export HS9001_DB_PATH="/home/db/history.sqlite"
```

## Usage
### Search

```
hs9001 search "term"
```

It is recommended to create an alias for search to make life easier, e. g.:

```
alias searchh='hs9001 search'
```

