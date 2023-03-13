package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"memegrab/cattp"
	"memegrab/sessions"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
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
	sessionLen := time.Now().Add(time.Hour * 720)
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

func getChannelMessages(botSession *discordgo.Session, conf *memeBotConf) []*discordgo.Message {
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

func getDbMessages[T []*FileInfo](db *sql.DB) T {
	query := `SELECT * FROM file_infos ORDER BY id ASC;`

	var files T

	rows, err := db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var filename string
		var sender string
		var timestamp time.Time
		// TODO: Recheck
		err = rows.Scan(&id, &filename, &sender, &timestamp)
		if err != nil {
			panic(err)
		}
		file := &FileInfo{
			ID:        id,
			FileName:  filename,
			Sender:    sender,
			Timestamp: timestamp}
		files = append(files, file)
	}
	if rows.Err() != nil {
		panic(err)
	}
	return files
}

func checkFileExists[T *FileInfo](db *gorm.DB, file T) bool {
	result := db.Limit(1).Find(&file)
	if result.Error != nil {
		panic(result.Error)
	}
	if result.RowsAffected < 1 {
		return false
	}
	return true
}

// TODO: Multiple files
func getMessageAttachment[T *FileInfo](message *discordgo.Message) T {
	for _, attach := range message.Attachments {
		res, err := http.Get(attach.URL)

		if err != nil {
			log.Println("Can't download attachment from URL")
			panic(err)
		}

		defer res.Body.Close()

		fileContent, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read %d bytes from Response Body\n", len(fileContent))

		file := &FileInfo{FileName: attach.Filename, Sender: message.Author.ID, Timestamp: message.Timestamp, Content: &fileContent}

		return file
	}
	return nil
}
