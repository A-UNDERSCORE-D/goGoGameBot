# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
## [Unreleased]
### Added
- Added a colour-stripped version of the message passed to chat bridge formats available under the name `MsgStripped`
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
[0.1.1]:      https://git.ferricyanide.solutions/A_D/goGoGameBot
[0.1.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot
[0.0.0]:      https://git.ferricyanide.solutions/A_D/goGoGameBot
