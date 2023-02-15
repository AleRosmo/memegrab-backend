package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type memeBot struct {
	id      string
	session *discordgo.Session
	conf    memeBotConf
	db      *sql.DB
}

type memeBotConf struct {
	token            string
	guildId          string
	observedChannels []string
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	log.Println("Loaded '.env' file")
}

func main() {
	var wg sync.WaitGroup

	conf := memeBotConf{
		token:   os.Getenv("BOT_TOKEN"),
		guildId: os.Getenv("BOT_GUILD_ID"),
		observedChannels: func() []string {
			chans := strings.Trim(strings.TrimSpace(os.Getenv("BOT_CHANNELS")), ":")
			if chans == "" {
				return []string{}
			}
			return strings.FieldsFunc(chans, func(r rune) bool {
				return r == ':' || r == ' '
			})
		}(),
	}
	botSession, err := discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("BOT_TOKEN")))
	if err != nil {
		panic(err)
	}
	memeBot := &memeBot{
		id:      "1",
		session: botSession,
		conf:    conf,
	}

	botSession.AddHandler(memeBot.messageHandler)

	defer wg.Done()
	wg.Add(1)
	err = botSession.Open()
	if err != nil {
		panic(err)
	}
	log.Println("Bot Started")
	log.Printf("Bot ID: %v\n", memeBot.id)

	dbConf := pgConf{
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	}
	db, err := pgInit(dbConf)
	defer db.Close()

	messages := getChannelMessages(botSession, conf)
	for _, msg := range messages {
		saveAttachment(msg.Attachments)
	}

	httpConf := httpConf{
		os.Getenv("HTTP_HOST"),
		os.Getenv("HTTP_PORT_PLAIN"),
		os.Getenv("HTTP_PORT_SECURE"),
		os.Getenv("HTTP_URL"),
		os.Getenv("HTTP_CERT"),
		os.Getenv("HTTP_KEY"),
	}

	err = startHTTPServer(httpConf)
	if err != nil {
		panic(err)
	}
	wg.Wait()

}

func (bot *memeBot) messageHandler(botSession *discordgo.Session, message *discordgo.MessageCreate) {
	log.Printf("New message from ID: %v\n", message.Author.ID)

	if strings.Contains(message.Content, "fanculo") {
		bot.session.ChannelMessageSend(message.ChannelID, "Ma si andiamo tutti affanculo")
	}

	if message.Author.ID == botSession.State.User.ID {
		return
	}
	if message.Attachments == nil {
		return
	}

	for _, obsId := range bot.conf.observedChannels {
		if message.ChannelID == obsId {
			if rand.Intn(2) == 0 {
				err := botSession.MessageReactionAdd(message.ChannelID, message.ID, "💾")
				if err != nil {
					log.Println("Error in adding reaction to message")
					return
				}
				log.Printf("Reacted with 💾 to message ID: %s\n", message.ID)
			} else {
				err := botSession.MessageReactionAdd(message.ChannelID, message.ID, "🍌")
				if err != nil {
					log.Println("Error in adding reaction to message")
					return
				}
				log.Printf("Reacted with 🍌 to message ID: %s\n", message.ID)
			}
			err := saveAttachment(message.Attachments)
			if err != nil {
				log.Println("Error in saving attachment file")
				return
			}
			log.Println("Saved message attachment")
		}
	}
}

func saveAttachment(attachments []*discordgo.MessageAttachment) error {
	for _, attach := range attachments {

		res, err := http.Get(attach.URL)

		if err != nil {
			log.Println("Can't download attachment from URL")
			return err
		}
		defer res.Body.Close()

		file, err := os.Create(filepath.Join("img", attach.Filename))
		if err != nil {
			log.Println("Error creating new file")
			return err
		}
		defer file.Close()
		written, err := io.Copy(file, res.Body)
		if err != nil {
			log.Println("Error writing to file")
			return err
		}
		log.Printf("Written %d bytes to file %s\n", written, attach.Filename)
	}
	return nil
}

func getChannelMessages(botSession *discordgo.Session, conf memeBotConf) []*discordgo.Message {
	channels, err := botSession.GuildChannels(conf.guildId)
	if err != nil {
		panic(err)
	}
	for _, ch := range channels {
		for _, obsId := range conf.observedChannels {
			if ch.ID == obsId {
				msg, err := botSession.ChannelMessages(ch.ID, 100, "", "", "")
				if err != nil {
					log.Println("Error in getting channel messages")
				}
				return msg
			}
		}
	}
	return nil
}

func getDbMessages(db *sql.DB, channels []string) {

}
