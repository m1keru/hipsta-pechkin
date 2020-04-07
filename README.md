## Little daemon seeding on email and transfer text from messages to given Telegram Chats
### HowTo
* Enable IMAP in your GMAIL
* Create Application password
* Create bot and get it's token 
* cp config.yml.tpl to config.yml and fill it.
* setup in screen or create systemd unit.

### How to build 

```bash
make
```

### How to add Channels

Add bot to given channel and check daemon output, it will print ChatID. Paste it into chats section in config.yml.

### Usefull links:

* [how to get application password](https://devanswers.co/outlook-and-gmail-problem-application-specific-password-required/)
* [How to register bot:](https://core.telegram.org/bots)

