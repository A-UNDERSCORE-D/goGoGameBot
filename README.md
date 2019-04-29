# GoGoGameBot
[![Go Report Card](https://goreportcard.com/badge/git.ferricyanide.solutions/A_D/goGoGamebot)](https://goreportcard.com/report/git.ferricyanide.solutions/A_D/goGoGamebot) [![Build Status](https://drone.ferricyanide.solutions/api/badges/A_D/goGoGameBot/status.svg)](https://drone.ferricyanide.solutions/A_D/goGoGameBot)

GoGoGameBot is a go reimplementation of my python GameBot. 
Both are designed to allow administration of game servers over IRC in a highly
customisable manner. 


## Config Format
GGGB's config is XML based and broken up into sections based on their areas. All config options fall into the main bot tag
and from there are broken into IRC and game configs. A example config is located in `doc/config.xml.example`. Further 
documentation for the config files is to be written
