package main

import (
	"database/sql"
	"fmt"
	"log"
	"memegrab/cattp"
	"memegrab/sessions"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

type DatabaseRequired interface {
	*sql.DB | *gorm.DB
}

// TODO: Declaring setter/getter
// type DiscordMessage interface {
// 	[]*discordgo.Message | *discordgo.MessageCreate
// 	get()
// }

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	log.Println("Loaded '.env' file")
}

func main() {
	var wg sync.WaitGroup

	conf := &memeBotConf{
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

	// Starting a new bot instance
	bot := New(conf)
	defer bot.db.Close()

	dbFiles := getDbMessages(bot.db)

	for _, file := range dbFiles {
		fmt.Printf("DB Info: %d - Filename: %s\n", file.ID, file.FileName)
	}

	messages := getChannelMessages(bot.discord, conf)

	for _, message := range messages {
		if len(message.Attachments) != 0 {
			file := getMessageAttachment(message)
			if file != nil {
				if !checkFileExists(bot.gorm, file) {
					log.Println("Not found on DB, saving")
					err := bot.saveAttachment(file)
					if err != nil {
						log.Println("Error in saving attachment file")
						return
					}
				}
			}
		}
	}

	// Get a session manager instance
	sessionLen := time.Hour * 720
	sessions := sessions.New(sessionLen)

	httpConf := cattp.Config{
		Host: os.Getenv("HTTP_HOST"),
		// os.Getenv("HTTP_PORT_PLAIN"),
		// os.Getenv("HTTP_PORT_SECURE"),
		Port: os.Getenv("HTTP_PORT_PLAIN"),
		// os.Getenv("HTTP_CERT"),
		//  os.Getenv("HTTP_KEY"),
		URL: os.Getenv("HTTP_URL"),
	}

	err := startWebApp(httpConf, bot.db, sessions)

	if err != nil {
		panic(err)
	}
	wg.Wait()
}
