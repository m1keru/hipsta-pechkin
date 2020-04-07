package main

import (
	"flag"
	"fmt"
	"github.com/bytbox/go-pop3"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

//EmailConfig config for email
type EmailConfig struct {
	ServerType    string `yaml:"servertype"`
	ServerAddress string `yaml:"serveraddress"`
	ServerPort    int64  `yaml:"serverport"`
	UserPassword  string `yaml:"userpassword"`
	UserAddress   string `yaml:"useraddress"`
	MailFolder    string `yaml:"mailfolder"`
}

//TelegramConfig struct for telegram
type TelegramConfig struct {
	Chats []int64 `yaml:"chats,flow"`
	Token string  `yaml:"token"`
}

// Config daemon config
type Config struct {
	TelegramConfig TelegramConfig `yaml:"telegramconfig"`
	EmailConfig    EmailConfig    `yaml:"emailconfig"`
}

func configRead(path *string) (Config, error) {
	configFile, err := os.Open(*path)
	if err != nil {
		log.Fatalf("Unable to read config file. Error:\n %v\n", err)
	}
	defer configFile.Close()
	var config Config
	decoder := yaml.NewDecoder(configFile)

	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Unable to Unmarshal Config, Error:\n %v\n", err)
	}
	return config, nil
}

func readEmailPop(config *Config) {
	server := config.EmailConfig.ServerAddress
	port := config.EmailConfig.ServerPort
	log.Println(port)
	login := config.EmailConfig.UserAddress
	password := config.EmailConfig.UserPassword
	popClient, err := pop3.DialTLS(fmt.Sprintf("%s:%d", server, port))
	if err != nil {
		log.Fatalf("unable to connect to mail server, Error: \n%v", err)
	}
	if err = popClient.Auth(login, password); err != nil {
		log.Fatalf("unable to auth at mail server, Error: \n%v", err)
	}
	msgs, _, err := popClient.ListAll()
	if err != nil {
		log.Println("Read it and generate password: https://devanswers.co/outlook-and-gmail-problem-application-specific-password-required/")
		log.Fatalf("unable to read list of emails at mail server, Error: \n%v", err)
	}
	for _, i := range msgs {
		text, _ := popClient.Retr(i)
		log.Println(text)
	}
}

func readEmailImap(config *Config, msgs *chan string) error {
	server := config.EmailConfig.ServerAddress
	port := config.EmailConfig.ServerPort
	login := config.EmailConfig.UserAddress
	password := config.EmailConfig.UserPassword
	folder := config.EmailConfig.MailFolder
	imapClient, err := client.DialTLS(fmt.Sprintf("%s:%d", server, port), nil)
	if err != nil {
		log.Fatalf("unable to connect to mail server, Error: \n%v", err)
	}
	defer imapClient.Logout()
	defer imapClient.Close()
	if err = imapClient.Login(login, password); err != nil {
		log.Fatalf("unable to auth at mail server, Error: \n%v", err)
	}

	mbox, err := imapClient.Select(folder, true)
	if err != nil {
		log.Println(err)
	}
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 3 {
		// We're using unsigned integers here, only substract if the result is > 0
		from = mbox.Messages - 3
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 20)
	done := make(chan error, 1)
	items := []imap.FetchItem{section.FetchItem(), imap.FetchFlags}
	go func() {
		done <- imapClient.Fetch(seqset, items, messages)

	}()

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	tmpSeqset := new(imap.SeqSet)

	for msg := range messages {
		skip := false
		for _, flag := range msg.Flags {
			if flag == "\\Seen" {
				skip = true
			}
		}
		if skip {
			continue
		}
		r := msg.GetBody(section)
		if r == nil {
			log.Printf("Server didn't returned message body\n")
			return err
		}
		mr, _ := mail.CreateReader(r)

		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Printf("Unable to read body: , Error:\n %v\n", err)
				tmpSeqset.AddNum(msg.SeqNum)
			}

			var out bool = false
			log.Println("PART")
			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				b, _ := ioutil.ReadAll(p.Body)
				log.Println("BODY:", string(b))
				*msgs <- string(b)
				tmpSeqset.AddNum(msg.SeqNum)
				out = true

			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Println("File attached but we are not FTP bro! filename: ", filename)
				tmpSeqset.AddNum(msg.SeqNum)
				out = true
				//*msgs <- filename
			}
			if out {
				break
			}
		}
	}
	if !tmpSeqset.Empty() {
		imapClient.Select(folder, false)
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []interface{}{imap.SeenFlag}
		err = imapClient.Store(tmpSeqset, item, flags, nil)
		if err != nil {
			log.Printf("Unable to mark as read, Error:\n %v\n", err)
		}
		log.Printf("Marked as read\n")
	}
	log.Println("circle")
	*msgs <- ""
	imapClient.Logout()
	imapClient.Close()
	return nil
}

func pollAndPushTelegram(config *Config, msgs *chan string) error {
	bot, err := tgbotapi.NewBotAPI(config.TelegramConfig.Token)
	if err != nil {
		log.Printf("Telegram: unable to connect, Error:\n %v", err)
		return err
	}

	chats := config.TelegramConfig.Chats
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10

	updates, err := bot.GetUpdatesChan(u)

	go func() {
		for {
			tmpMsg := <-*msgs
			for _, chat := range chats {
				ms := tgbotapi.NewMessage(chat, tmpMsg)
				bot.Send(ms)
				log.Println("Message sent to ", chat)
			}
			time.Sleep(time.Second * 10)
		}
	}()

	for update := range updates {

		if update.Message == nil { // ignore any non-Message Updates
			log.Printf("Telegram:%v", update)
			continue
		}
		log.Printf("Telegram: user: [%s]  message: %s chatId: %d", update.Message.From.UserName, update.Message.Text, update.Message.Chat.ID)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		bot.Send(msg)
	}
	return nil
}

func main() {
	configFilePath := flag.String("config", "config.yml", "path to config file")

	flag.Parse()
	config, err := configRead(configFilePath)
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}

	//readEmailPop(&config)

	msgs := make(chan string)
	tgmsg := make(chan string, 10)

	go pollAndPushTelegram(&config, &tgmsg)
	for {
		go func() {
			if err := readEmailImap(&config, &msgs); err != nil {
				log.Printf("ImapError: %v", err)
			}
		}()

		msg := <-msgs
		if msg != "" {
			println("TEXT:", msg)
			tgmsg <- msg
		}

		time.Sleep(time.Second * 5)
	}
}
