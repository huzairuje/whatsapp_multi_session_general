package commandhandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"whatsapp_multi_session_general/primitive"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	Clients = make(map[string]*whatsmeow.Client)
)

type Message struct {
	MessageID string
	Jid       string
	Type      string
	Body      string
	Sent      bool
	FileName  string
}

type CommandHandler struct {
	Container *sqlstore.Container
}

func NewCommandHandler(container *sqlstore.Container) CommandHandler {
	return CommandHandler{
		Container: container,
	}
}

func (ch CommandHandler) NewHandleCheckUser(sender types.JID, args []string) (response []types.IsOnWhatsAppResponse) {
	fmt.Printf("Checking users: %v", args)
	if len(args) < 1 {
		fmt.Errorf("Usage: checkuser <phone numbers...>")
		return nil
	}

	resp, err := Clients[sender.User].IsOnWhatsApp(args)
	if err != nil {
		fmt.Errorf("Failed to check if users are on WhatsApp: %v", err)
		return nil
	}

	for _, item := range resp {
		logMessage := fmt.Sprintf("%s: on WhatsApp: %t, JID: %s", item.Query, item.IsIn, item.JID)

		if item.VerifiedName != nil {
			logMessage += fmt.Sprintf(", business name: %s", item.VerifiedName.Details.GetVerifiedName())
		}
		fmt.Printf(logMessage)
		response = append(response, item)
	}
	return response
}

func (ch CommandHandler) HandleSendNewTextMessage(sender types.JID, textMsg string, jid string) (messageID string, err error) {
	recipient, ok := parseJID(jid)
	if !ok {
		return
	}

	msg := &waProto.Message{
		Conversation: proto.String(textMsg),
	}

	fmt.Printf("Sending message to %s: %s", recipient, msg.GetConversation())

	//set message id from std lib whatsmeo
	messageID = Clients[sender.User].GenerateMessageID()

	fmt.Printf("request messageID from sendRequestExtra is : %v ", messageID)

	resp, err := Clients[sender.User].SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		fmt.Errorf("Error sending message: %v", err)
		return
	}

	Clients[sender.User].AddEventHandler(EventHandler)

	fmt.Printf("response messageID is : %v ", resp.ID)

	fmt.Printf("Message sent (server timestamp: %s)", resp.Timestamp)
	return resp.ID, nil
}

func (ch CommandHandler) HandleSendNewTextMessageBulk(sender types.JID, textMsg string, jids []string) {
	var wg sync.WaitGroup
	for _, jid := range jids {
		wg.Add(1)
		go func(jid string) {
			defer wg.Done()

			recipient, ok := parseJID(jid)
			if !ok {
				return
			}

			msg := &waProto.Message{
				Conversation: proto.String(textMsg),
			}

			fmt.Printf("Sending message to %s: %s", recipient, msg.GetConversation())

			_, err := Clients[sender.User].SendMessage(context.Background(), recipient, msg)
			if err != nil {
				fmt.Errorf("Error sending message: %v", err)
				return
			}

			fmt.Printf("Message sent (server timestamp: %s)", time.Now())
		}(jid)
	}
	wg.Wait()
}

func (ch CommandHandler) GetSingleQR(ctx context.Context, clients map[string]*whatsmeow.Client, senderJidTypes types.JID) (string, error) {
	device, err := ch.Container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	// Create a client for each device
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(device, clientLog)
	client.AddEventHandler(EventHandler)

	// Connect the client synchronously
	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			panic(err)
		}
		err = client.Connect()
		if err != nil {
			panic(err)
		}

		fmt.Println("Waiting for QR code or login event...")
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// Add the client to the map
				clients[senderJidTypes.User] = client
				image, errGenerateCode := generateQRCode(evt.Code)
				if errGenerateCode != nil {
					// Log the error for debugging
					fmt.Println("Error generating QR code:", errGenerateCode)
					return "", errGenerateCode
				}
				return string(image), nil
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err := client.Connect()
		if err != nil {
			panic(err)
		}
		// Add the client to the map
		clients[device.ID.User] = client
		return "", nil
	}

	return "", nil
}

func (ch CommandHandler) GetSpecificQR(ctx context.Context, clients map[string]*whatsmeow.Client, jid types.JID) (string, error) {
	devices, err := ch.Container.GetAllDevices()
	if err != nil {
		panic(err)
	}

	var device *store.Device
	if len(devices) > 0 {
		for _, val := range devices {
			jidUser := strings.TrimSpace(jid.User)
			valIdUser := strings.TrimSpace(val.ID.User)

			if jidUser == valIdUser {
				device = val
			}
		}
	}

	if device == nil || device.ID.User == "" {
		device = ch.Container.NewDevice()
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(device, clientLog)
	client.AddEventHandler(EventHandler)

	// Connect the client synchronously
	if client.Store.ID == nil {
		qrChan, errGetQr := client.GetQRChannel(ctx)
		if errGetQr != nil {
			panic(errGetQr)
		}
		err = client.Connect()
		if err != nil {
			panic(err)
		}

		fmt.Println("Waiting for QR code or login event...")
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// Add the client to the map
				clients[jid.User] = client
				image, errGenerateCode := generateQRCode(evt.Code)
				if errGenerateCode != nil {
					// Log the error for debugging
					fmt.Println("Error generating QR code:", errGenerateCode)
					return "", errGenerateCode
				}
				return string(image), nil
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err := client.Connect()
		if err != nil {
			panic(err)
		}
		// Add the client to the map
		clients[jid.User] = client
		return "", nil
	}
	return "", nil
}

func createImageMessage(uploaded whatsmeow.UploadResponse, data *[]byte, captionMsg string) *waProto.Message {
	return &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Url:           proto.String(uploaded.URL),
			Mimetype:      proto.String(http.DetectContentType(*data)),
			Caption:       &captionMsg,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(*data))),
			MediaKey:      uploaded.MediaKey,
			FileEncSha256: uploaded.FileEncSHA256,
			DirectPath:    proto.String(uploaded.DirectPath),
		},
	}
}

func createVideoMessage(uploaded whatsmeow.UploadResponse, data *[]byte, captionMsg string) *waProto.Message {
	return &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			Url:           proto.String(uploaded.URL),
			Mimetype:      proto.String(http.DetectContentType(*data)),
			Caption:       &captionMsg,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(*data))),
			MediaKey:      uploaded.MediaKey,
			FileEncSha256: uploaded.FileEncSHA256,
			DirectPath:    proto.String(uploaded.DirectPath),
		},
	}
}

func createAudioMessage(uploaded whatsmeow.UploadResponse, data *[]byte) *waProto.Message {
	return &waProto.Message{
		AudioMessage: &waProto.AudioMessage{ // Change ImageMessage to AudioMessage
			Url:           proto.String(uploaded.URL),
			Mimetype:      proto.String(http.DetectContentType(*data)),
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(*data))),
			MediaKey:      uploaded.MediaKey,
			FileEncSha256: uploaded.FileEncSHA256,
			DirectPath:    proto.String(uploaded.DirectPath),
		},
	}
}

func createDocumentMessage(fileName string, uploaded whatsmeow.UploadResponse, data *[]byte, captionMsg string) *waProto.Message {
	return &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			FileName:      proto.String(fileName),
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(*data)),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(*data))),
			Title:         proto.String(fmt.Sprintf("%s%s", "document", filepath.Ext(uploaded.URL))),
			Caption:       &captionMsg,
		},
	}
}

func ValidateStringArrayAsStringArray(stringInput string) ([]string, error) {
	// Validate if the string is empty
	if strings.TrimSpace(stringInput) == "" {
		return nil, errors.New("empty recipients")
	}

	// Split the string by commas
	stringsArr := strings.Split(stringInput, ",")

	// Validate and clean each substring
	var stringSlice []string
	for _, str := range stringsArr {
		trimmedStr := strings.TrimSpace(str)
		// Append the converted value to the result slice
		stringSlice = append(stringSlice, trimmedStr)
	}

	return stringSlice, nil
}

func NewHandleSendImage(sender types.JID, JIDS []string, data []byte, captionMsg string) ([]Message, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var sliceM []Message
	var errs []error

	for _, jid := range JIDS {
		wg.Add(1)
		go func(jid string) {
			defer wg.Done()

			recipient, ok := parseJID(jid)
			if !ok {
				mu.Lock()
				errs = append(errs, fmt.Errorf("invalid JID: %s", jid))
				mu.Unlock()
				return
			}

			uploaded, err := Clients[sender.User].Upload(context.Background(), data, whatsmeow.MediaImage)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to upload file: %v", err))
				mu.Unlock()
				return
			}

			msg := createImageMessage(uploaded, &data, captionMsg)
			resp, err := Clients[sender.User].SendMessage(context.Background(), recipient, msg)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("error sending image message: %v", err))
				mu.Unlock()
				return
			}

			fmt.Printf("Message sent (server timestamp: %s)\n", resp.Timestamp)

			m := Message{resp.ID, recipient.String(), "media", "", true, ""}
			mu.Lock()
			sliceM = append(sliceM, m)
			mu.Unlock()
		}(jid)
	}

	wg.Wait()

	// Handle errors if any
	if len(errs) > 0 {
		return nil, errs[0] // You might want to handle multiple errors differently
	}

	return sliceM, nil
}

func NewHandleSendDocument(sender types.JID, JID []string, fileName string, data []byte, captionMsg string) ([]Message, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var sliceM []Message
	var errs []error

	for _, jid := range JID {
		wg.Add(1)
		go func(jid, fileName string) {
			defer wg.Done()

			recipient, ok := parseJID(jid)
			if !ok {
				mu.Lock()
				errs = append(errs, fmt.Errorf("invalid JID: %s", jid))
				mu.Unlock()
				return
			}

			uploaded, err := Clients[sender.User].Upload(context.Background(), data, whatsmeow.MediaDocument)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to upload file: %v", err))
				mu.Unlock()
				return
			}

			msg := createDocumentMessage(fileName, uploaded, &data, captionMsg)
			resp, err := Clients[sender.User].SendMessage(context.Background(), recipient, msg)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("error sending document message: %v", err))
				mu.Unlock()
				return
			}

			fmt.Printf("Message sent (server timestamp: %s)\n", resp.Timestamp)

			m := Message{resp.ID, recipient.String(), "media", "", true, fileName}
			mu.Lock()
			sliceM = append(sliceM, m)
			mu.Unlock()
		}(jid, fileName)
	}

	wg.Wait()

	// Handle errors if any
	if len(errs) > 0 {
		return nil, errs[0] // You might want to handle multiple errors differently
	}

	return sliceM, nil
}

func NewHandleSendVideo(sender types.JID, JID []string, data []byte, captionMsg string) ([]Message, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var sliceM []Message
	var errs []error

	for _, jid := range JID {
		wg.Add(1)
		go func(jid string) {
			defer wg.Done()

			recipient, ok := parseJID(jid)
			if !ok {
				mu.Lock()
				errs = append(errs, fmt.Errorf("invalid JID: %s", jid))
				mu.Unlock()
				return
			}

			uploaded, err := Clients[sender.User].Upload(context.Background(), data, whatsmeow.MediaImage)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to upload file: %v", err))
				mu.Unlock()
				return
			}

			msg := createVideoMessage(uploaded, &data, captionMsg)
			resp, err := Clients[sender.User].SendMessage(context.Background(), recipient, msg)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("error sending image message: %v", err))
				mu.Unlock()
				return
			}

			fmt.Printf("Message sent (server timestamp: %s)\n", resp.Timestamp)

			m := Message{resp.ID, recipient.String(), "media", "", true, ""}
			mu.Lock()
			sliceM = append(sliceM, m)
			mu.Unlock()
		}(jid)
	}

	wg.Wait()

	// Handle errors if any
	if len(errs) > 0 {
		return nil, errs[0] // You might want to handle multiple errors differently
	}

	return sliceM, nil
}

func NewHandleSendAudio(sender types.JID, JID []string, data []byte) ([]Message, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var sliceM []Message
	var errs []error

	for _, jid := range JID {
		wg.Add(1)
		go func(jid string) {
			defer wg.Done()

			recipient, ok := parseJID(jid)
			if !ok {
				mu.Lock()
				errs = append(errs, fmt.Errorf("invalid JID: %s", jid))
				mu.Unlock()
				return
			}

			uploaded, err := Clients[sender.User].Upload(context.Background(), data, whatsmeow.MediaImage)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to upload file: %v", err))
				mu.Unlock()
				return
			}

			msg := createAudioMessage(uploaded, &data)
			resp, err := Clients[sender.User].SendMessage(context.Background(), recipient, msg)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("error sending image message: %v", err))
				mu.Unlock()
				return
			}

			fmt.Printf("Message sent (server timestamp: %s)\n", resp.Timestamp)

			m := Message{resp.ID, recipient.String(), "media", "", true, ""}
			mu.Lock()
			sliceM = append(sliceM, m)
			mu.Unlock()
		}(jid)
	}

	wg.Wait()

	// Handle errors if any
	if len(errs) > 0 {
		return nil, errs[0] // You might want to handle multiple errors differently
	}

	return sliceM, nil
}

// Parse a JID from a string. If the string starts with a +, it is removed.
func parseJID(arg string) (types.JID, bool) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), true
	} else {
		recipient, err := types.ParseJID(arg)
		if err != nil {
			fmt.Errorf("Invalid JID %s: %v", arg, err)
			return recipient, false
		} else if recipient.User == "" {
			fmt.Errorf("Invalid JID %s: no server specified", arg)
			return recipient, false
		}
		return recipient, true
	}
}

func EventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())

	//handling on receipt
	case *events.Receipt:
		// Get the jid
		recipientJID := v.Chat
		messageIDs := v.MessageIDs
		fmt.Println("recipientJID ", recipientJID)
		fmt.Println("messageIDs ", messageIDs)

		for _, singleMessageID := range v.MessageIDs {
			// Get the type of the event
			// sender, Delivered, TypeRead
			evtType := v.Type.GoString()
			if strings.Contains(evtType, "sender") {
				evtType = "Sent"
				fmt.Printf("MessageID %v is sent ", singleMessageID)

			} else if strings.Contains(evtType, "Delivered") {
				evtType = "Delivered"
				fmt.Printf("MessageID %v is Delivered ", singleMessageID)
			} else if strings.Contains(evtType, "Read") {
				evtType = "Read"
				fmt.Printf("MessageID %v is Read ", singleMessageID)
			}
		}
		break

	default:
		_ = evt
		break
	}
}

func generateQRCode(code string) ([]byte, error) {
	// Create QR code image
	qrImage, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}

	return qrImage, nil
}

func (ch CommandHandler) AutoLogin() {
	devices, err := ch.Container.GetAllDevices()
	if err != nil {
		fmt.Errorf("err sqlstore.New : %v ", err)
		return
	}

	if len(devices) > 0 {
		for _, val := range devices {
			if val.ID.User != "" {
				device := val
				//set new client
				clientLog := waLog.Stdout("Client", "DEBUG", true)
				client := whatsmeow.NewClient(device, clientLog)
				client.AddEventHandler(EventHandler)

				// Connect the client synchronously
				if client.Store.ID != nil {
					err := client.Connect()
					if err != nil {
						fmt.Errorf("err sqlstore.New : %v ", err)
						return
					}
					Clients[val.ID.User] = client
				}
			} else {
				continue
			}
		}
	}
	return
}

func (ch CommandHandler) AutoDisconnect() {
	devices, err := ch.Container.GetAllDevices()
	if err != nil {
		fmt.Errorf("err sqlstore.New : %v ", err)
		return
	}

	if len(devices) > 0 {
		for _, val := range devices {
			if val.ID.User != "" {
				device := val
				//set new client
				clientLog := waLog.Stdout("Client", "DEBUG", true)
				client := whatsmeow.NewClient(device, clientLog)
				client.AddEventHandler(EventHandler)

				// Connect the client synchronously
				if client.Store.ID != nil {
					client.Disconnect()
				}
			} else {
				continue
			}
		}
	}
	return
}

func (ch CommandHandler) AutoLogOut() {
	devices, err := ch.Container.GetAllDevices()
	if err != nil {
		fmt.Errorf("err sqlstore.New : %v ", err)
		return
	}

	if len(devices) > 0 {
		for _, val := range devices {
			if val.ID.User != "" {
				device := val
				//set new client
				clientLog := waLog.Stdout("Client", "DEBUG", true)
				client := whatsmeow.NewClient(device, clientLog)
				client.AddEventHandler(EventHandler)

				// Connect the client synchronously
				if client.Store.ID != nil {
					err := client.Logout()
					if err != nil {
						fmt.Errorf("err sqlstore.New : %v ", err)
					}
				}
			} else {
				continue
			}
		}
	}
	return
}

func (ch CommandHandler) GetAllDevices() (response []primitive.Devices) {
	container, err := ch.Container.GetAllDevices()
	if err != nil {
		fmt.Errorf("Failed to check if users are on WhatsApp: %v", err)
		return nil
	}

	if len(container) > 0 {
		for _, item := range container {
			newItem := primitive.Devices{
				PushName:   item.PushName,
				Platform:   item.Platform,
				User:       item.ID.User,
				Server:     item.ID.Server,
				IsLoggedIn: Clients[item.ID.User].IsLoggedIn(),
			}
			response = append(response, newItem)
		}
	} else {
		emptyResp := make([]primitive.Devices, 0)
		response = emptyResp
	}
	return response
}
