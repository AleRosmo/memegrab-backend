package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math/rand"
	"memegrab/cattp"
	"memegrab/sessions"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _testGorm *gorm.DB

type DatabaseRequired interface {
	*sql.DB | *gorm.DB
}

type memeBot struct {
	discord *discordgo.Session
	// sessions sessions.SessionManager
	conf memeBotConf
	db   *sql.DB // ! DEPRECATE
	gorm *gorm.DB
}

// TODO: Saved files in properties for memeBot, duplicate files
func (bot *memeBot) saveDbMessage(file *FileInfo) (*FileInfo, error) {
	query := `INSERT INTO public.saved (file_name) VALUES ($1) RETURNING id;`
	var id int

	err := bot.db.QueryRow(query, file.FileName).Scan(&id)
	if err != nil {
		return nil, err
	}
	file.ID = id
	log.Printf("Saved to DB %s with ID %d\n", file.FileName, id)

	return file, nil
}

type memeBotConf struct {
	token            string
	guildId          string
	observedChannels []string
}

type FileInfo struct {
	ID       int    `gorm:"primaryKey"`
	FileName string `json:"file_name,omitempty"`
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

	defer wg.Done()
	wg.Add(1)
	err = botSession.Open()
	if err != nil {
		panic(err)
	}
	log.Println("Bot Started")
	// log.Printf("Bot ID: %v\n", memeBot.id)

	dbConf := pgConf{
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	}

	db, err := pgInit(dbConf)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	dbGorm, err := testInitGorm(dbConf)
	if err != nil {
		panic(err)
	}
	memeBot := &memeBot{
		discord: botSession,
		conf:    conf,
		db:      db,
		gorm:    dbGorm,
	}

	botSession.AddHandler(memeBot.messageHandler)

	sessionLen := time.Now().Add(time.Hour * 720)
	sessions := sessions.New(sessionLen)

	dbFiles := getDbMessages(db)
	for _, file := range dbFiles {
		fmt.Printf("DB Info: %d - Filename: %s\n", file.ID, file.FileName)
	}

	messages := getChannelMessages(botSession, conf)
	for _, msg := range messages {
		memeBot.saveAttachment(msg.Attachments)
	}

	httpConf := cattp.Config{
		Host: os.Getenv("HTTP_HOST"),
		// os.Getenv("HTTP_PORT_PLAIN"),
		// os.Getenv("HTTP_PORT_SECURE"),
		Port: os.Getenv("HTTP_PORT_PLAIN"),
		// os.Getenv("HTTP_CERT"),
		//  os.Getenv("HTTP_KEY"),
		URL: os.Getenv("HTTP_URL"),
	}

	err = startWebApp(httpConf, db, sessions)
	if err != nil {
		panic(err)
	}
	wg.Wait()
}

func (bot *memeBot) messageHandler(botSession *discordgo.Session, message *discordgo.MessageCreate) {
	log.Printf("New message from ID: %v\n", message.Author.ID)

	if strings.Contains(message.Content, "fanculo") {
		bot.discord.ChannelMessageSend(message.ChannelID, "Ma si andiamo tutti affanculo")
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
			err := bot.saveAttachment(message.Attachments)
			if err != nil {
				log.Println("Error in saving attachment file")
				return
			}
			log.Println("Saved message attachment")
		}
	}
}

// TODO: Avoid logging duplicates (for double posts)
// TODO: Check file signature ?????????? in SHA to avoid duplicated files
// * SHURI SUGGESTS:
// * To verify it's indetical get the lenght, split in 2048 bytes chunks and compare each chunk
func (bot *memeBot) saveAttachment(attachments []*discordgo.MessageAttachment) error {
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

		// f := &FileInfo{FileName: attach.Filename}
		// f, err = bot.saveDbMessage(f)
		// if err != nil {
		// 	return err
		// }
		// log.Printf("Saved file %s in DB with ID %d\n", f.FileName, f.ID)

		f := FileInfo{FileName: attach.Filename}
		tx := bot.gorm.Table("public.saved").
			Clauses(clause.Returning{}).
			Create(&f)

		savedFile := tx.Statement.Dest.(*FileInfo)
		// GORM TEST
		log.Printf("Saved file %s in DB with ID %d WITH GORM\n", f.FileName, savedFile.ID)
		// var gormFileRead FileInfo
		// _testGorm.Table("public.saved").First(&gormFileRead, "file_name = ?", attach.Filename)
		// fmt.Println(gormFile)
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

func getDbMessages[T []*FileInfo](db *sql.DB) T {
	query := `SELECT * FROM public.saved;`

	var files T

	rows, err := db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var filename string
		// TODO: Recheck
		err = rows.Scan(&id, &filename)
		if err != nil {
			panic(err)
		}
		info := &FileInfo{id, filename}
		files = append(files, info)
	}
	if rows.Err() != nil {
		panic(err)
	}
	return files
}
