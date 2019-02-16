# Missing features
- [ ] Automated backups--Feature creep?
- [ ] Automated restarts--Feature creep?
- [ ] Inter-game communication, so that games can talk between each-other while IRC is down

# Other stuff
- [ ] Config documentation
- [x] General cleanup
- [ ] Game's privmsg hook should get an ID for unloading later
    - [ ] Bot's hook commands should all return UIDs for later unloading
- [ ] Fix GetHostStats util function returning 0% for total system CPU usage
- [ ] Add bot memory and CPU usage to stats command
- [ ] Add restart command. Yay execve fun
- [ ] Losing the connection to IRC should not be an issue for the bot. It should do the following instead:
    - [ ] Warn all running games that IRC's connection has been lost
    - [ ] Try to reconnect to IRC repeatedly, notifying only when its reconnected
        - [ ] When it DOES reconnect, it should state in the channel which games are running
