

## `lines`


```shell
$ lines -c ls --title ' Lines: $PWD/$TEXT' -s '@ less $TEXT'
```

```shell
$ lines -c 'echo -e "YES\nNO"' --title 'A question?' -s '> [ "$TEXT" == "YES" ] \n Exit'
```

```shell
if $(lines -c 'echo -e "YES\nNO"' --title 'A question?' -s '> [ "$TEXT" == "YES" ] \n Exit') ; then
    # user answered yes...
    ...
else
    ...
fi
```

