package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// TriviaTrainerTelegramAPIURL is Telegram's API URL for the TriviaTrainerBot, as defined in
	//  'docs/telegram_config.md'
	TriviaTrainerTelegramAPIURL = "https://api.telegram.org/bot373437281:AAGB3ai1jxjEdrWVaHztvORuEUVLw8VwVcw/"
	// TriviaTrainerLocalEndpoint is the local endpoint as set for the TriviaTrainerBot, as defined in
	//  'docs/telegram_config.md'
	TriviaTrainerLocalEndpoint = "/bot373437281:AAGB3ai1jxjEdrWVaHztvORuEUVLw8VwVcw"

	// TriviaTrainerLocalPort is the local port to serve on, as set for the TriviaTrainerBot and defined in
	//  'docs/telegram_config.md'
	TriviaTrainerLocalPort = 88
)

// Session keeps the state of a particular session
type Session struct {
	TelegramAPIURL string
	LocalEndpoint  string
	LocalPort      int
	FullChainPath  string
	PrivateKeyPath string
	HandlerFunc    func(chan *HookHitJSON, Session)
	handlerChannel chan *HookHitJSON
}

// HookHitJSON is the JSON sent by Telegram when a message is delivered
type HookHitJSON struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int    `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

// SendMessageJSON is the JSON that must be sent to the Telegram API to successfully send a message
type SendMessageJSON struct {
	ChatID    int    `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"` // This must be empty, Markdown, or HTML
}

// Serve kicks off server, which will call the caller's specified handler function
func (sess *Session) Serve() error {
	// Create channel for new requests
	sess.handlerChannel = make(chan *HookHitJSON)
	// Kick off delegate thread to handle requests
	go sess.HandlerFunc(sess.handlerChannel, *sess)

	http.HandleFunc(sess.LocalEndpoint, sess.hookHitHandler)
	portString := fmt.Sprintf(":%v", sess.LocalPort)
	err := http.ListenAndServeTLS(portString, sess.FullChainPath, sess.PrivateKeyPath, nil)
	return err
}

func (sess Session) hookHitHandler(res http.ResponseWriter, req *http.Request) {
	var telegramData HookHitJSON

	err := json.NewDecoder(req.Body).Decode(&telegramData)
	if err != nil {
		sess.handlerChannel <- nil
		return
	}
	sess.handlerChannel <- &telegramData
}

// SendMessage allows the caller to send a specified message to a particular chatID
func (sess Session) SendMessage(chatID int, message string) error {
	requestData := SendMessageJSON{chatID, string(message), "Markdown"}
	requestJSON, err := json.Marshal(requestData)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("GET", sess.TelegramAPIURL+"sendMessage", bytes.NewReader(requestJSON))
	if err != nil {
		return errors.New("Request failed in request building stage")
	}

	request.Header.Add("Content-Type", "application/json")

	requestClient := http.Client{
		Timeout: time.Second * 10,
	}

	response, err := requestClient.Do(request)
	if err != nil {
		return errors.New("Request failed in transit to Telegram")
	}

	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.New("Request failed in response JSON reading stage")
	}

	if response.StatusCode != 200 {
		errorText := "Request failed with HTTP error " + string(response.StatusCode)
		return errors.New(errorText)
	}

	// Good job! We didn't have any errors
	return nil
}
