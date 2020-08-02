// Discord IPC Client - currently only works on Windows builds of Discord since we only connect using the named pipe
// This is a bit reverse engineered using some older resources on GitHub and parts of the Discord Game SDK source code
// Most functions contain abstractions to make use a little bit easier
//
// Credits to the helpful resources I used:
// - https://github.com/k3rn31p4nic/discoIPC/blob/master/discoIPC/ipc.py#L60-L83
// - https://gist.github.com/lun-4/d21f9634c1514fd469f29c1b6e5ab5d8
// - https://github.com/hugolgst/rich-go

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/microsoft/go-winio"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

type Handshake struct {
	Version  string `json:"v"`
	ClientID string `json:"client_id"`
}

type Activity struct {
	Details string `json:"details"`
	State   string `json:"state"`
	Assets  Assets `json:"assets"`
}

// DiscordRawResponse contains a parsed version of the data from Discord's IPC responses
// This shouldn't really be used directly, but rather used by each exported function to form a nicer Type that's
// friendlier to work with so the main program doesn't need to fool unmarshalling data
type DiscordRawResponse struct {
	Opcode int32  // Response code from Discord
	Length int32  // Length of JSON payload after header
	Data   []byte // Payload minus the header
	Valid  bool   // Indicates if Data looks like valid JSON data
}

func (r *DiscordRawResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.Data, v)
}

// Assets for Rich Presence, used by SetActivity()
type Assets struct {
	LargeImage string `json:"large_image"`
	LargeText  string `json:"large_text"`
	SmallImage string `json:"small_image"`
	SmallText  string `json:"small_text"`
}

// Command sent by us to Discord IPC. Shouldn't really only be used by client functions
type Frame struct {
	Command   string    `json:"cmd"`
	Arguments Arguments `json:"args"`
	Nonce     []byte    `json:"nonce"`
}

type Arguments struct {
	Pid      int      `json:"pid"`
	Activity Activity `json:"activity"`
}

// DiscordResponse is the standard JSON payload format from the Discord IPC server
type DiscordResponse struct {
	Version string              `json:"v"`     // seems to be unused but possibly indicates the RPC server version
	Command string              `json:"cmd"`   // Discord-specific command
	Data    DiscordResponseData `json:"data"`  // JSON object that varies based on the command
	Event   string              `json:"evt"`   // Discord-specific event
	Nonce   string              `json:"nonce"` // The same Nonce that was sent in the client side request
}

type DiscordResponseUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Bot           bool   `json:"bot"`
	Flags         int    `json:"flags"`
	PremiumType   int    `json:"premium_type"`
}

type DiscordResponseConfig struct {
	CDNHost     string `json:"cdn_host"`
	APIEndpoint string `json:"api_endpoint"`
	Environment string `json:"environment"`
}

type DiscordResponseData struct {
	Config *DiscordResponseConfig `json:"config"`
	User   *DiscordResponseUser   `json:"user"`
}

// DiscordIPC represents the main IPC client type
type DiscordIPC struct {
	Version    string
	ClientID   string
	connection net.Conn
	connected  bool
}

// Login to Discord with the set client ID and version
// Assumes the client is already connected
func (client *DiscordIPC) Login() (*DiscordResponse, error) {

	if client.Version == "" || client.ClientID == "" {
		return nil, errors.New("client values Version and ClientID must both be set")
	}

	if !client.connected {
		return nil, errors.New("client must call Connect() first")
	}

	payload, err := json.Marshal(Handshake{
		Version:  client.Version,
		ClientID: client.ClientID,
	})

	if err != nil {
		return nil, err
	}

	raw, err := client.Send(0, payload)
	if err != nil {
		return nil, err
	}

	var response DiscordResponse
	err = raw.JSON(&response)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// SetActivity uses the RichPresence API to set Discord Rich Presence
// See the Activity type for the available properties
func (client *DiscordIPC) SetActivity(activity Activity) (*DiscordResponse, error) {

	payload, err := json.Marshal(Frame{
		Command: "SET_ACTIVITY",
		Arguments: Arguments{
			Pid:      os.Getpid(),
			Activity: activity,
		},
		Nonce: Nonce(),
	})

	if err != nil {
		log.Fatalln("Error trying to Marshal activity payload: ", err.Error())
		return nil, err
	}

	rawResponse, err := client.Send(1, payload)

	if err != nil {
		log.Println("SetActivity() failed: ", err.Error())
		return nil, err
	}

	var response DiscordResponse
	err = rawResponse.JSON(&response)
	if err != nil {
		log.Println("Failed to Unmarshal SetActivity() response")
		return nil, err
	}

	return &response, nil
}

// Read forms a DiscordRawResponse from the connected Discord RPC socket.
func (client *DiscordIPC) Read() (*DiscordRawResponse, error) {

	// because we don't get an io.EOF when we get to the end of the data, we have to perform 2 reads here. read the 8
	// byte header to determine the opcode, and more importantly the length of the JSON payload, then make a 2nd buffer
	// which is the size of the JSON payload and read that too. then validate the payload as JSON data

	header := make([]byte, 8) // 8 comes from the 8 byte header: opcode + payload length
	n, err := client.connection.Read(header)

	if err != nil {
		return nil, err
	}

	if n != 8 {
		return nil, errors.New("expected 8 byte header from discord ipc, got " + string(n) + " byte(s) instead")
	}

	headerBuffer := bytes.NewReader(header)
	response := &DiscordRawResponse{}

	// read Opcode to determine what kind of response this is. e.g, Opcode 2 usually means error
	err = binary.Read(headerBuffer, binary.LittleEndian, &response.Opcode)
	if err != nil {
		return nil, err
	}

	// read buffer Length to determine how much more we need to read
	err = binary.Read(headerBuffer, binary.LittleEndian, &response.Length)

	if err != nil {
		return nil, err
	}

	// read remaining data which should just be valid JSON data at this point
	response.Data = make([]byte, response.Length)
	n, err = client.connection.Read(response.Data)

	if err != nil {
		return nil, err
	}

	// can be removed later, just a handy way to make sure the Read completed as intended
	response.Valid = json.Valid(response.Data)

	log.Println("R:", string(response.Data), "OPCODE:", response.Opcode)

	return response, nil
}

// Write() invokes Write() on the IPC connection with a required opcode and JSON encoded payload. The payload length is
// automatically written to the stream first
func (client *DiscordIPC) Write(opcode int32, payload []byte) error {
	buffer := new(bytes.Buffer)

	err := binary.Write(buffer, binary.LittleEndian, opcode)
	if err != nil {
		return err
	}

	err = binary.Write(buffer, binary.LittleEndian, int32(len(payload)))
	if err != nil {
		return err
	}

	log.Println("S:", string(payload))
	buffer.Write(payload)
	_, err = client.connection.Write(buffer.Bytes())

	if err != nil {
		return err
	}

	return nil
}

/// Send() invokes Write() then immediately calls Read()
func (client *DiscordIPC) Send(opcode int32, payload []byte) (*DiscordRawResponse, error) {

	err := client.Write(opcode, payload)
	if err != nil {
		return nil, err
	}

	response, err := client.Read()

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Invokes ConnectToInstance() with 0 as the default instance
func (client *DiscordIPC) Connect() error {
	return client.ConnectToInstance("0")
}

func (client *DiscordIPC) ConnectToInstance(instance string) error {
	timeout := time.Second * 2
	cx, err := winio.DialPipe(`\\.\pipe\discord-ipc-`+instance, &timeout)

	if err != nil {
		return err
	}

	client.connection = cx
	client.connected = true

	return nil
}

func (client *DiscordIPC) Disconnect() {
	_ = client.connection.Close()
	client.connected = false
}

func Nonce() []byte {
	nonce := make([]byte, 12)

	if _, err := rand.Read(nonce); err != nil {
		panic(err.Error()) // something is very very wrong
	}

	return nonce
}
