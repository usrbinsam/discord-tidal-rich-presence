package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDiscordIPC_Connect(t *testing.T) {

	discord := DiscordIPC{
		Version:  "1",
		ClientID: os.Getenv("CLIENT_ID"),
	}

	err := discord.Connect()

	if err != nil {
		t.Error("Connecting to Discord IPC failed. Is Discord running? Error: ", err.Error())
	}

	discord.Disconnect()
}

func TestNonce(t *testing.T) {
	nonce1 := Nonce()
	nonce2 := Nonce()

	if len(nonce1) < 12 {
		t.Error("Expected nonce to be be at least length 12, got length: ", len(nonce1))
	}

	if string(nonce1) == string(nonce2) {
		t.Errorf("2 calls to Nonce() returned the same value. %q == %q", nonce1, nonce2)
	}
}

func TestDiscordIPC_Login(t *testing.T) {
	discord := DiscordIPC{
		Version:  "1",
		ClientID: os.Getenv("CLIENT_ID"),
	}

	err := discord.Connect()

	if err != nil {
		t.Error("Connect() failed. Is Discord running? Error: ", err.Error())
	}

	response, err := discord.Login()

	if err != nil {
		t.Error("Login() failed. Error: ", err.Error())
	}

	if response.Event != "READY" {
		t.Error("Login() response expected Event READY, got:", response.Event)
	}

	discord.Disconnect()
}

func TestActivity_JSON(t *testing.T) {
	act := Activity{
		Details: "Playing some super sweaty game ...",
		State:   "Playing a game",
		Assets:  Assets{},
	}
	payload := act.JSON()

	var check Activity
	err := json.Unmarshal(payload, &check)

	if err != nil {
		t.Error("Failed to Unmarshal an Activity:", err.Error())
	}

	if check.Assets.SmallText != "none" {
		t.Error(`Assets.SmallText didn't get set to "none"`)
	}
	if check.Assets.SmallImage != "none" {
		t.Error(`Assets.SmallImage didn't get set to "none"`)
	}
	if check.Assets.LargeText != "none" {
		t.Error(`Assets.LargeText didn't get set to "none"`)
	}
	if check.Assets.LargeImage != "none" {
		t.Error(`Assets.LargeImage didn't get set to "none"`)
	}
}

func TestDiscordIPC_SetActivity(t *testing.T) {
	discord := DiscordIPC{
		Version:  "1",
		ClientID: os.Getenv("CLIENT_ID"),
	}

	err := discord.Connect()

	if err != nil {
		t.Error("Connect() failed. Is Discord running? Error: ", err.Error())
	}

	if err != nil {
		t.Error("Login() failed. Error:", err.Error())
	}

	retval, err := discord.SetActivity(Activity{
		Details: "Being gay simulator",
		State:   "Being gay",
	})

	if err != nil {
		t.Errorf("SetActivity() failed. Error: %q Discord: %v", err.Error(), retval)
	}
}
