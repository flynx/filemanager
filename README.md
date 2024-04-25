

## `lines`


```shell
$ lines -c ls --title ' Lines: $PWD/$TEXT' -s '@ less $TEXT'
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

