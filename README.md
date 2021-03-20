Add this to .bashrc

if [ -n "$PS1" ] ; then
    PROMPT_COMMAND='hs9001 add "$(history 1)"'
fi
