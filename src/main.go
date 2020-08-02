package main

import (
	"log"
	"os"
	"strings"
	"time"
)

func setNowPlaying(ipc *DiscordIPC, song, artist string) {

	if artist != "" {
		artist = "by " + artist
	}
	_, _ = ipc.SetActivity(Activity{
		Details: song,
		State:   artist,
		Assets: Assets{
			LargeText:  "none",
			SmallText:  "none",
			LargeImage: "none",
			SmallImage: "none",
		},
	})

}

func main() {

	discord := &DiscordIPC{
		Version:  "1",
		ClientID: os.Getenv("CLIENT_ID"),
	}

	// repeatedly try to connect to Discord until its running
	for {
		log.Println("Attempting Discord connection ...")
		err := discord.Connect()

		if err != nil {
			log.Println("Error connecting to Discord. Is Discord running? Error:", err.Error())
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	log.Println("Logging in to Discord ...")
	response, err := discord.Login()
	if err != nil {
		log.Fatal("Failed to login to discord: ", err.Error())
	}

	log.Printf("Login response: %#v", response)

	if response.Event != "READY" {
		log.Fatalln("Handshake with Discord failed.")
	}

	var lastSong string

	for {
		song, err := WindowTitle("TIDAL.exe")

		if err != nil {
			log.Println("error getting Tidal song:", err.Error())
		}

		// tidal may have stopped
		if song == "" && lastSong != "" {
			setNowPlaying(discord, "", "")
		} else if song != lastSong {
			np := strings.Split(song, " - ")
			if len(np) == 2 {
				setNowPlaying(discord, np[0], np[1])
			}
			lastSong = song
		}

		time.Sleep(time.Second * 5)
	}
}
