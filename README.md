

## `lines`


Basic browser/viewer
```shell
$ DIR=`pwd` \
    lines \
        -c 'echo "../"; ls -F --group-directories-first "$DIR"' \
        -k F3:'@ [[ "$TEXT_LEFT" =~ \/$ ]] || less "$DIR/${TEXT_LEFT%[*/=>@|]}"' \
        -s 'DIR=> [[ "$TEXT_LEFT" =~ \/$ ]] && realpath -s "$DIR/${TEXT_LEFT}" || realpath -s "$DIR" \n Update' \
        --title ' $DIR %SPAN' 
        -b --span-separator="│" \
```



```shell
$ lines -c ls --title ' Lines: $PWD/$TEXT' -s '@ less $TEXT'
```

```shell
$ ls | lines \
    --size '80%,80%' \
    --theme background:black:black \
    --title 'Lines: $PWD/$TEXT$SELECTED' \
    -s '@ lines "$TEXT" --title "View: ./$TEXT"'
```

```shell
$ ls -alFh --group-directories-first \
    | sed \
        -e '1,2d' \
        -e 's/ *$//' \
        -e 's/^\(.*[0-9]\{2\}:[0-9]\{2\}\) \(.*\)$/\2%SPAN \1/' \
        -e 's/\(%SPAN \)\([^ ]* \)\([^ ]* \)\{3\}/\1\2/' \
    | lines \
        -s '@ [[ "$LEFT_TEXT" =~ \/$ ]] || less "$LEFT_TEXT"' \
        --key Delete:'@ scriots/yesOrNo "Delete: $TEXR_LEFT?" && rm "$TEXR_LEFT" || true' \
        --span-separator="│" \
        --title " $TEXT_LEFT %SPAN" \
        -f "lines.go" \
        --border \
        --selection 'grep "\.go"'
```

```shell
$ lines \
    --title 'A question?' \
    -c 'echo -e "YES\nNO"' \
    -r "Fail" \
    -s '> [ "$TEXT" == "YES" ] \n Exit'
```

```shell
if $(lines \
        --title 'A question?' \
        -c 'echo -e "YES\nNO"' \
        -r "Fail" \
        -s '> [ "$TEXT" == "YES" ] \n Exit') ; then
    # user answered yes...
    ...
else
    ...
fi
```

