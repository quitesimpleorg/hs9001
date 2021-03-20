# hs9001
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
    PROMPT_COMMAND='hs9001 add "$(history 1)"'
fi
```

## Usage
### Search
```
hs9001 search "term"
```

Is is recommended to create an aias for search to make life easier. 
