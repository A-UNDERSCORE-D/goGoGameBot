# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
## [Unreleased]
### Added
- Different games running under the bot can now have messages passed between them
### Fixed
- unset formatters for GameRegexps and various other configs no longer error on start or in use.
- Passing messages between games no longer causes an error when the other games are not running

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
[0.2.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/c54e1526b5d97e5f7e9ed7c0412e1164bb0c04cb...05443765e782d1b7aa0220fc9309755b28ffa11e
[0.1.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/b27ecee11a0add85feb208210c07419d42d4a97d...c54e1526b5d97e5f7e9ed7c0412e1164bb0c04cb
[0.1.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot/compare/673bce90c9a03f2cc7c3d0cd7005bf06a0bfafa6...b27ecee11a0add85feb208210c07419d42d4a97d
[0.0.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot
