package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	Name       string
	tables     []string
	connection *sql.DB
}

var db Database

func CreateDB(name string) *Database {
	conn, err := sql.Open("sqlite", name)
	if err != nil {
		fmt.Printf("Error opening database: %v", err)
	}

	err = conn.Ping()
	if err != nil {
		fmt.Printf("Error connecting to database: %v", err)
	}
	fmt.Println("Database connected successfully.")

	_, err = conn.Exec(`
	CREATE TABLE IF NOT EXISTS silenced (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		discord_id TEXT NOT NULL,
		guild_id TEXT NOT NULL,
		silenced_until DATETIME NOT NULL
	)`)
	if err != nil {
		fmt.Printf("Error creating table %v", err)
		return nil
	}
	fmt.Println("Table created or already exists")

	db = Database{
		Name:       name,
		tables:     []string{},
		connection: conn,
	}

	return &db
}

func GetDB() Database {
	if db.connection == nil {
		fmt.Println("DB not initialized")
	}
	return db
}

func (db *Database) InsertSilence(userId string, guildId string, minutes int) {
	stmt, err := db.connection.Prepare("INSERT INTO silenced (discord_id, guild_id, silenced_until) VALUES (?, ?, ?)")
	if err != nil {
		fmt.Printf("Error preparing statement: %v", err)
		return
	}
	defer stmt.Close()

	endTime := time.Now().Add(time.Duration(minutes) * time.Minute)

	res, err := stmt.Exec(userId, guildId, endTime.Format("2006-01-02 15:04:05"))
	if err != nil {
		fmt.Println("Error executing insert statement")
	}

	fmt.Println(res)
}

func (db *Database) DeleteOldSilences() {
	stmt, err := db.connection.Prepare("DELETE FROM silenced WHERE silenced_until < ?")
	if err != nil {
		fmt.Printf("Error preparing statement %v", err)
		return
	}
	defer stmt.Close()

	now := time.Now()

	res, err := stmt.Exec(now.Format("2006-01-02 15:04:05"))
	if err != nil {
		fmt.Println("Error executing delete statement")
	}
	fmt.Println(res)
}

func (db *Database) IsUserSilenced(discordId string) bool {
	var exists bool

	err := db.connection.QueryRow("SELECT EXISTS (SELECT 1 FROM silenced WHERE discord_id = ?)", discordId).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		fmt.Printf("Error querying db: %v", err)
		return false
	}
	return exists
}

func (db *Database) Close() {
	db.connection.Close()
}
