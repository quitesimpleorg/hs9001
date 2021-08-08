# hs9001
hs9001 (history search 9001) is an easy, quite simple bash history enhancement. It simply writes all
your bash commands into an sqlite database. You can then search this database.

It improves over bash's built-in history mechanism as it'll aggregate the shell history of all open shells,
timestamp them and also record additional information such as the directory a command was executed in.

## Usage / Examples
### Search

```
hs [search terms]
```
You can further filter with options like `-cwd`, `-after` and so on...
For a full list, see `-help`.

```
hs -cwd . 
``` 
Lists all commands ever entered in this directory

```
hs -after yesterday -cwd . git
``` 
Lists all git commands in the current directory which have been entered today.

Also, it (by default) replaces bash's built-in CTRL-R mechanism, so hs9001's database will be used instead
of bash's limited history files. 

## Install

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

### From source
```
go build
#move hs9001 to a PATH location
```

## Setup / Config

Add this to .bashrc

```
eval "$(hs9001 bash-enable)"
```

This will also create a `hs`alias so you have to type less in everyday usage.

By default, every system user gets his own database. You can override this by setting the environment variable
for all users that should write to your unified database.
```
export HS9001_DB_PATH="/home/db/history.sqlite"
```


