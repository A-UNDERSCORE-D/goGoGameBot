# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## Fixed

- The Minecraft transformer no longer causes latent formatting issues when the first part of a message is formatted

## Changed

- TODO: stuff about the updated transports and configs

## Added

- There is now a reconnect command to force a reconnection to the chat platform

## [0.4.7] - 2019-11-24

### Fixed

- Simple transformers no longer misbehave when given an empty replace

## [0.4.6] - 2019-11-24

### Added

- A new transformer is available under the `minecraft` name. Mapped messages will be in the form of minecraft JSON.
Use this with `/tellraw @a` for a quick and easy minecraft bridge that supports full colours, formatting, and URLs

### Fixed

- Game stdio is now run though transformers before they are processed
- IRC no longer quits instantly after attempting to reconnect due to lag issues

## [0.4.5] - 2019-11-10

### Fixed

- Games no longer sometimes send IRC formatting codes to each other
- Games sending to other games now verify that the target game is running, rather than themselves

### Changed

- The external format on games now use the same formatting tools that the rest of the formats use
meaning that you can now map etc messages from other games. (which should be setup to use intermediate formats when
transferring messages)

## [0.4.4] - 2019-11-06

### Fixed

- Newlines and Carriage Returns are correctly escaped by the escape method
- Games that are not running no longer show as stopping when the bot is stopped

### Changed

- newlines and carriage returns are now escaped
- Raw access to IRC lines is essentially removed
- Non existent (and/or empty) formatters are now valid for incoming-to-game messages (eg `<message>`). If a formatter is
empty or doesn't exist, anything that uses it will simply return
- Bot control commands (start, stop, restart) now require admin level 3, and status requires no admin, and game raw now requires an admin of 3

### Added

- You can now specify ENV variables (and specifically avoid the system ENV vars) when launching games, use `<environment>`
to add (or override) environment variables. If you want to explicitly ignore the system environment variables, add
`dont_copy_env="true"` to the attributes on `<game>`.
- A new formatting system that uses characters similar to the old one (except for colour), but is far
more extensible. The new system is based on "Transformers" each of which behave differently, but work by converting strings to
an "Intermediate Form". For example, the IRC formatter (which is local to the IRC implementation) converts IRC formatting and
colour to the intermediate form
- Continuing to the above, the new Transformers available everywhere are:

| Name     | Description |
| -------- | -------- |
| Strip  | Does what it says on the tin. Strips intermediate form |
| Simple | Does simple text transformations (find and replace) for "standard" formatting, and will convert colour to the given palette and then replace the colour with the given string|

- Pre Roll regexp config option. This option allows you to transform every incoming line from stdin and stderr with a regex
- The IRC Conn now maintains its own ping to the server, which will disconnect it after 30 seconds

## [0.4.2] - 2019-08-29

### Added

- reimplemented status command. It now also supports specifying game names to get status specific to each game. In future
games will be able to add additional information to status, by way of extra stdin commands and regex
- Continued implementation of format/transformer

## [0.4.1] - 2019-08-23

### Fixed

- Some commands were skipped by the IRC side of the bot, meaning they never reached the game to
be displayed

## [0.4.0] - 2019-08-23

### Added

- A new bool is available on game regexps that sends the result of the regex to the local
game as well as any other options. Note that these strings are sent through the `external`
formatter already available on the game.
- A new function called stripAll is available in all format strings. This function allows
you to strip **all** control codes from a string, both IRC control codes and ASCII control
codes (namely those below ascii `0x20`)
- A new function called `eat` is available on all format strings. It will eat any number of
arguments of any type, and return an empty string
- `Storage` is now available on most message formatters, (any that accept a `dataForFmt`).
It allows for cross-invocation storage of values. For use in formatters

### Fixed

- Various (recovered) panics caused by parting/quitting/kicking users without a message
- Control codes being shared between games without being stripped when sent over the external
line interface

### Changed

- Reimplemented all of the IRC side of the bot
- GGGB is no longer tied to an IRC connection. IRC connections can be dropped without running servers being stopped
- Other connection types other than IRC are now experimentally supported. Some things are still IRC-centric, namely, colour formatting
- Due to the above, the config syntax has completely changed, please validate configs against the new XSD before attempting
to start under the new version

## [0.3.5] - 2019-07-01

### Fixed

- Messages from other games are no longer sent directly to stdin on the target game without going through the external formatter

## [0.3.4] - 2019-07-01

### Fixed

- `stdio_regexp` incorrectly named `stdio_regexps`

## [0.3.3] - 2019-07-01

### Added

- The reload command has returned. It allows for runtime game config reloading
- Formats on `<stdio_regexp>` entries now have access to the above defined templates
- Channels that games reference will be automatically joined when the bot is started.
Note that this does not apply to reloaded configs. Any channels added after a reload will need to be manually joined

### Fixed

- Command line interface commands no longer require a command prefix to be executed
- Game inter-communication no longer tries to send messages to games that are not running

### Changed

- `<regexp>` in game is now `<stdio_regexp>`, and the control elements other than `<regexp>` and `<format>` have been moved
to attributes.

### Removed

- Vendored dependencies

## [0.3.2] - 2019-06-30

### Fixed

- Regexps for games were incorrectly named in the config

## [0.3.1] - 2019-06-30

### Fixed

- Working directories are no longer ignored when explicitly set

## [0.3.0] - 2019-06-27

### Added

- Stats command now shows the current goroutine count and the current go runtime version
- reimplemented command system, subcommands are now supported (one level only) as well as help for commands
- reimplemented game config, and game system in general.
- game templates are now linked and can reference each other, additionally an arbitrary number of additional templates
may be defined for reference in automatically called ones

### Fixed

- SASL authentication no longer sends an additional `\x00` in the PLAIN auth string
- `game.CompileOrError` function no longer discards the passed function mapping

## [0.2.3] - 2019-04-13

### Added

- The format for incoming lines from other games has access to a new function, `mapColours`, it allows you to map the raw
IRC colours in the given string to the set colour map on the game
- All format strings have access to a new function `stripColours` that allows them to strip raw colours from the given line

### Fixed

- colour escapes are now evaluated in format strings

### Changed

- irc-go updated to latest version

## [0.2.2] - 2019-04-01

### Added

- Games can now be automatically restarted when they exit with exit code 0

## [0.2.1] - 2019-03-29

### Added

- Different games running under the bot can now have messages passed between them
- Stopping a game that is not running now results in a message stating that the game cannot be stopped
- restart command (will not work in Delve)

### Fixed

- unset formatters for GameRegexps and various other configs no longer error on start or in use.
- Passing messages between games no longer causes an error when the other games are not running
- included GameRegexps now include past the first entry in the `regexps` tag
- stopping games with the stopgame command no-longer leaves the game in an un-startable state
- starting games that are already started now returns a cleaner error message

## [0.2.0] - 2019-02-22

### Added

- A colour-stripped version of the message passed to chat bridge formats available under the name `MsgStripped`
- Join/Part forward formats. They exist in game configs as `JoinPartFormat` and have a bool available to check whether or not it is a join or a part

### Changed

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

- Changed CHANGELOG type

### Fixed

- Status command when used in a terminal no longer has missing format string warnings
- Startup welcome message is now correctly formatted

## [0.1.1] - 2019-02-13

### Added

- Added auto-start to games
- Added bot memory usage to status command

### Changed

- Made starting with a nonexistent config create the default one and print a message mentioning this

## [0.1.0] - 2019-02-14

### Added

- Added gggb version
- Added a few info logs in various places

## [0.0.0] - 2019-01-01

### Added

- Many undocumented additions

[Unreleased]: https://git.ferricyanide.solutions/A_D/goGoGameBot
[0.4.7]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/41e314f473987a85db1bff74aecbe2b878dc6462...0563e36d876550fa2e77413f2f43cbedb6389663
[0.4.6]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/121712681fb22f0099251f83249bb8451b58f212...41e314f473987a85db1bff74aecbe2b878dc6462
[0.4.5]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/933d526afe8f7c31d0c809d03ec2182b8d205d51...121712681fb22f0099251f83249bb8451b58f212
[0.4.4]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/7d083518df663bd8cd7b71f525c2c3624b69dfb7...933d526afe8f7c31d0c809d03ec2182b8d205d51
[0.4.2]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/9c16e5bdef8c6382b72b20f3713ab4a4d45883ff...7d083518df663bd8cd7b71f525c2c3624b69dfb7
[0.4.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/f9f59d1c07c93a54019804a22df4feef23409dca...9c16e5bdef8c6382b72b20f3713ab4a4d45883ff
[0.4.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/9ab227b07856945179e05159c48fc0bb08025efa...f9f59d1c07c93a54019804a22df4feef23409dca
[0.3.5]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/88f7d651928e613dc57fa0e8d5b0de2cc970fc6d...9ab227b07856945179e05159c48fc0bb08025efa
[0.3.4]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/b38b8c345c4f614ff167ceff17ea75a3d477aca0...88f7d651928e613dc57fa0e8d5b0de2cc970fc6d
[0.3.3]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/7853b0a8ac7fe63fc9be5e671ffbcfe209a2e3c3...b38b8c345c4f614ff167ceff17ea75a3d477aca0
[0.3.2]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/059e4fc266c88b2b877892ff6fe3c27703c28428...7853b0a8ac7fe63fc9be5e671ffbcfe209a2e3c3
[0.3.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/cb2ad2488fdb8c2ff69080a567777bdc113dd780...059e4fc266c88b2b877892ff6fe3c27703c28428
[0.3.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/e150762e9da3b0c48f4688610fe78c17aee1595d...cb2ad2488fdb8c2ff69080a567777bdc113dd780
[0.2.3]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/3b8f793144078472c44c4874e3ab0db1c6d6ffe4...e150762e9da3b0c48f4688610fe78c17aee1595d
[0.2.2]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/d7bd61c31ff1bfb051c527866b0e64d3b434dac4...3b8f793144078472c44c4874e3ab0db1c6d6ffe4
[0.2.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/05443765e782d1b7aa0220fc9309755b28ffa11e...d7bd61c31ff1bfb051c527866b0e64d3b434dac4
[0.2.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/c54e1526b5d97e5f7e9ed7c0412e1164bb0c04cb...05443765e782d1b7aa0220fc9309755b28ffa11e
[0.1.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/b27ecee11a0add85feb208210c07419d42d4a97d...c54e1526b5d97e5f7e9ed7c0412e1164bb0c04cb
[0.1.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/673bce90c9a03f2cc7c3d0cd7005bf06a0bfafa6...b27ecee11a0add85feb208210c07419d42d4a97d
[0.0.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot
