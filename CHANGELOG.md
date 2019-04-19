# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
## [Unreleased]

### Added
- Stats command now shows the current goroutine count and the current go runtime version

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
- stopping games with the stopgame command no-longer leaves the game in an unstartable state
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
[0.2.3]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/3b8f793144078472c44c4874e3ab0db1c6d6ffe4...e150762e9da3b0c48f4688610fe78c17aee1595d
[0.2.2]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/d7bd61c31ff1bfb051c527866b0e64d3b434dac4...3b8f793144078472c44c4874e3ab0db1c6d6ffe4
[0.2.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/05443765e782d1b7aa0220fc9309755b28ffa11e...d7bd61c31ff1bfb051c527866b0e64d3b434dac4
[0.2.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/c54e1526b5d97e5f7e9ed7c0412e1164bb0c04cb...05443765e782d1b7aa0220fc9309755b28ffa11e
[0.1.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/b27ecee11a0add85feb208210c07419d42d4a97d...c54e1526b5d97e5f7e9ed7c0412e1164bb0c04cb
[0.1.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/673bce90c9a03f2cc7c3d0cd7005bf06a0bfafa6...b27ecee11a0add85feb208210c07419d42d4a97d
[0.0.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot
