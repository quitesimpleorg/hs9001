# hs9001
hs9001 (history search 9001) is an easy, quite simple bash history enhancement. It simply writes all
your bash commands into an sqlite database. You can then search this database.


## Install

### From source
```
go build
#move hs9001 to a PATH location
```

### Debian / Ubuntu
Latest release can be installed using apt
```
curl -s https://repo.quitesimple.org/repo.quitesimple.org.asc | sudo apt-key add -
echo "deb https://repo.quitesimple.org/debian/ default main" | sudo tee /etc/apt/sources.list.d/quitesimple.list
sudo apt-get update
sudo apt-get install hs9001
```

### Alpine
```
wget https://repo.quitesimple.org/repo%40quitesimple.org-5f3d101.rsa.pub -O /etc/apk/repo@quitesimple.org-5f3d101.rsa.pub
echo "https://repo.quitesimple.org/alpine/quitesimple/" >> /etc/apk/repositories
apk update
apk add hs9001
```


### Setup / Config

```
hs9001 init
```

Add this to .bashrc

```
if [ -n "$PS1" ] ; then
    PROMPT_COMMAND='hs9001 add -ret $? "$(history 1)"'
fi
```
By default, every system user gets his own database. You can override this by setting the environment variable
for all users that should write to your unified database.

```
export HS9001_DB_PATH="/home/db/history.sqlite"
```

## Usage
### Search

```
hs9001 search [search terms]
```

It is recommended to create an alias for search to make life easier, e. g.:

```
alias searchh='hs9001 search'
```

