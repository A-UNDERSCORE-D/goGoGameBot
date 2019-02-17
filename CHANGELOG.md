# 0.2.0
- Changed behaviour of config files when it comes to game commands.
 ```xml
 <command name="raw" requires_admin="true" stdin_format="{{.ArgString}}"/>
 ```
 is now:
 ```xml
<command name="raw" requires_admin="true">
<format>{{.ArgString}}</format>
</command>
```
With format being the "standard" formatter with all its available tools and settings.

- Added a colour-stripped version of the message passed to chat bridge formats available under the name `MsgStripped`
# 0.1.1
- Added auto-start to games
- Made starting with a nonexistent config create the default one and print a message mentioning this
- Added bot memory usage to status command
# 0.1.0
- Added gggb version
- Added a few info logs in various places
