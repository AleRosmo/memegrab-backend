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
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func New(botConfig *memeBotConf) *memeBot {
	// Create new Discord session
	botSession, err := discordgo.New(fmt.Sprintf("Bot %s", os.Getenv("BOT_TOKEN")))
	if err != nil {
		panic(err)
	}

	// Create websocket with discord
	err = botSession.Open()
	if err != nil {
		panic(err)
	}

	log.Println("Bot Started")
	// log.Printf("Bot ID: %v\n", memeBot.id)

	// Create DB configuration
	dbConf := pgConf{
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	}

	// Start default PostgreSQL connector
	db, err := pgInit(dbConf)
	if err != nil {
		panic(err)
	}

	// Start GORM DB connector
	dbGorm, err := testInitGorm(dbConf)
	if err != nil {
		panic(err)
	}

	// Instance reference to bot context
	memeBot := &memeBot{
		discord: botSession,
		conf:    botConfig,
		db:      db,
		gorm:    dbGorm,
	}

	// Add Handler for messages
	botSession.AddHandler(memeBot.messageHandler)

	return memeBot
}

type memeBot struct {
	discord *discordgo.Session
	conf    *memeBotConf
	db      *sql.DB
	gorm    *gorm.DB
}

func (bot *memeBot) messageHandler(session *discordgo.Session, message *discordgo.MessageCreate) {
	log.Printf("New message from %v (%v)\n", message.Member.Nick, message.Author.ID)

	// if strings.Contains(message.Content, "fanculo") {
	// 	bot.discord.ChannelMessageSend(message.ChannelID, "Ma si andiamo tutti affanculo")
	// }

	if message.Author.ID == session.State.User.ID {
		return
	}
	if len(message.Attachments) == 0 {
		return
	}

	isObservedChannel := slices.Contains(bot.conf.observedChannels, message.ChannelID)

	if !isObservedChannel {
		return
	}

	if rand.Intn(2) == 0 {
		err := session.MessageReactionAdd(message.ChannelID, message.ID, "💾")
		if err != nil {
			log.Println("Error in adding reaction to message")
			return
		}
		log.Printf("Reacted with 💾 to message ID: %s\n", message.ID)
	} else {
		err := session.MessageReactionAdd(message.ChannelID, message.ID, "🍌")
		if err != nil {
			log.Println("Error in adding reaction to message")
			return
		}
		log.Printf("Reacted with 🍌 to message ID: %s\n", message.ID)
	}

	file := getMessageAttachment(message.Message)
	exist := checkFileExists(bot.gorm, file)
	if exist {
		return
	}
	log.Println("Not found on DB, saving")
	err := bot.saveAttachment(file)
	if err != nil {
		log.Println("Error in saving attachment file")
		return
	}
	log.Println("Saved message attachment")
}

func (bot *memeBot) saveAttachment(file *FileInfo) error {
	localFile, err := os.Create(filepath.Join("img", file.FileName))
	if err != nil {
		log.Println("Error creating new file")
		return err
	}
	written, err := localFile.Write(*file.Content)
	if err != nil {
		log.Println("Error writing to file")
		return err
	}
	log.Printf("Written %d bytes to file %s\n", written, localFile.Name())

	// Gorm updates our custom type instance with the ID returned
	tx := bot.gorm.
		Clauses(clause.Returning{}).
		Omit("Content").
		Create(&file)

	if tx.Error != nil {
		panic(err)
	}
	// GORM TEST
	log.Printf("Saved file %s in DB with ID %d WITH GORM\n", file.FileName, file.ID)
	// var gormFileRead FileInfo
	// _testGorm.Table("file_info").First(&gormFileRead, "file_name = ?", attach.Filename)
	// fmt.Println(gormFile)
	return nil
}

type memeBotConf struct {
	token            string
	guildId          string
	observedChannels []string
}

type FileInfo struct {
	ID        int       `gorm:"primaryKey"`
	FileName  string    `gorm:"file_name"`
	Sender    string    `gorm:"sender"`
	Timestamp time.Time `gorm:"timestamp"`
	Content   *[]byte   `gorm:"-"`
}

func checkFileExists(db *gorm.DB, file *FileInfo) bool {
	result := db.Omit("Content").Where(&file).Find(&file)
	if result.Error != nil {
		panic(result.Error)
	}
	if result.RowsAffected < 1 {
		return false
	}
	return true
}

// TODO: Multiple files
func getMessageAttachment(message *discordgo.Message) *FileInfo {
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

func getDbMessages(db *sql.DB) []*FileInfo {
	query := `SELECT * FROM file_infos ORDER BY id ASC;`

	var files []*FileInfo

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
