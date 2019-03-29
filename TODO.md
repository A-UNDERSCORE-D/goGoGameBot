# Missing features
- [ ] Automated backups--Feature creep?
- [ ] Automated restarts--Feature creep?
- [ ] Make failed sasl auth actually do passwd based nickserv auth
- [ ] Message after a reload stating that something happened
- [ ] sane defaults for a good number of the format strings

# Other stuff
- [ ] the Status command should take an optional list of games to drop statuses for
- [ ] The string "$c" is stripped when sent to games. Investigate
- [ ] Config documentation--separate markdown file
- [ ] Game's privmsg hook should get an ID for unloading later
    - [ ] Bot's hook commands should all return UIDs for later unloading
- [ ] Fix GetHostStats util function returning 0% for total system CPU usage
- [ ] Add restart command. Yay execve fun
- [ ] Losing the connection to IRC should not be an issue for the bot. It should do the following instead:
    - [ ] Warn all running games that IRC's connection has been lost
    - [ ] Try to reconnect to IRC repeatedly, notifying only when its reconnected
        - [ ] When it DOES reconnect, it should state in the channel which games are running
