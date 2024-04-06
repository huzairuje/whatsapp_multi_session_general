package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"whatsapp_multi_session_general/commandhandler"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types"
)

type Handler struct {
	CommandHandler commandhandler.CommandHandler
}

func NewHandler(commandhandler commandhandler.CommandHandler) Handler {
	return Handler{
		CommandHandler: commandhandler,
	}
}

// ServeSendText handles sending text messages
func (h Handler) ServeSendText(c *gin.Context) {
	// Get query parameters
	senderString := c.Query("sender")

	if senderString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "sender seharusnya diisi dengan nomor yang valid"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	clientSpecificUser := commandhandler.Clients[senderJidTypes.User]
	if clientSpecificUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "gagal kirim"})
		return
	}

	if clientSpecificUser.IsLoggedIn() {
		var msgBody struct {
			Recipient string `json:"recipient" binding:"required"`
			Message   string `json:"message" binding:"required"`
		}

		if err := c.BindJSON(&msgBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error decoding JSON"})
			return
		}

		msgID, err := h.CommandHandler.HandleSendNewTextMessage(senderJidTypes, msgBody.Message, msgBody.Recipient)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success", "id_pesan": msgID})
		return
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{"message": "gagal kirim, tolong hit endpoint untuk melakukan qrcode"})
}

// ServeSendTextBulk handles sending bulk text messages
func (h Handler) ServeSendTextBulk(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	// Get query parameters
	senderString := c.Query("sender")
	if senderString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "sender seharusnya diisi dengan nomor yang valid"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	clientSpecificUser := commandhandler.Clients[senderJidTypes.User]
	if clientSpecificUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "gagal kirim"})
		return
	}

	if clientSpecificUser.IsLoggedIn() {
		var msgBody struct {
			Recipients []string `json:"recipients" binding:"required"`
			Message    string   `json:"message" binding:"required"`
		}

		if err := c.BindJSON(&msgBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "error decoding JSON"})
			return
		}

		h.CommandHandler.HandleSendNewTextMessageBulk(senderJidTypes, msgBody.Message, msgBody.Recipients)

		c.JSON(http.StatusOK, gin.H{"message": "success"})
		return
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{"message": "gagal kirim, tolong hit endpoint untuk melakukan qrcode"})
}

// ServeStatus returns the current status of the client
func (h Handler) ServeStatus(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	// Get query parameters
	senderString := c.Query("sender")
	if senderString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "sender should be filled"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	clientSpecificUser := commandhandler.Clients[senderJidTypes.User]
	if clientSpecificUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	if clientSpecificUser.IsLoggedIn() {
		response := struct {
			ID       string `json:"id"`
			PushName string `json:"pushName"`
			IsLogin  bool   `json:"isLogin"`
		}{
			ID:       commandhandler.Clients[senderJidTypes.User].Store.ID.String(),
			PushName: commandhandler.Clients[senderJidTypes.User].Store.PushName,
			IsLogin:  commandhandler.Clients[senderJidTypes.User].IsLoggedIn(),
		}

		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"message": "gagal kirim, tolong hit endpoint untuk melakukan qrcode"})
}

// ServeAllDevices checks user status
func (h Handler) ServeAllDevices(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	response := h.CommandHandler.GetAllDevices()
	c.JSON(http.StatusOK, response)
	return
}

// ServeCheckUser checks user status
func (h Handler) ServeCheckUser(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	// Get query parameters
	senderString := c.Query("sender")
	if senderString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "sender should be filled"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	clientSpecificUser := commandhandler.Clients[senderJidTypes.User]
	if clientSpecificUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	if clientSpecificUser.IsLoggedIn() {
		var msgBody struct {
			Recipients []string `json:"recipients" binding:"required"`
		}

		if err := c.BindJSON(&msgBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error decoding JSON"})
			return
		}

		response := h.CommandHandler.NewHandleCheckUser(senderJidTypes, msgBody.Recipients)

		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"message": "gagal kirim, tolong hit endpoint untuk melakukan qrcode kembali"})
}

func (h Handler) NewUploadHandler(c *gin.Context) {
	// Get query parameters
	senderString := c.Query("sender")
	if senderString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender should be filled"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	clientSpecificUser := commandhandler.Clients[senderJidTypes.User]
	if clientSpecificUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if clientSpecificUser.IsLoggedIn() {
		err := c.Request.ParseMultipartForm(10 << 20)
		if err != nil {
			handleError(c.Writer, http.StatusBadRequest, "Failed to parse multipart form", err)
			return
		}

		// Get the files
		files, ok := c.Request.MultipartForm.File["file"]
		if !ok || len(files) == 0 {
			handleError(c.Writer, http.StatusBadRequest, "No files found in the request", nil)
			return
		}

		recipientJIDs := c.Request.FormValue("recipients")
		captionMsg := c.Request.FormValue("caption")

		var resp []commandhandler.Message

		for _, handler := range files {
			// Open the file
			file, err := handler.Open()
			if err != nil {
				handleError(c.Writer, http.StatusInternalServerError, "Failed to open file", err)
				return
			}
			defer file.Close()

			// Read the file data
			data, err := io.ReadAll(file)
			if err != nil {
				handleError(c.Writer, http.StatusInternalServerError, "Failed to read file data", err)
				return
			}

			sliceJID, err := commandhandler.ValidateStringArrayAsStringArray(recipientJIDs)
			if err != nil {
				handleError(c.Writer, http.StatusInternalServerError, "Something went wrong with parameter jid", err)
				return
			}

			var uploadResp []commandhandler.Message
			mimeType := http.DetectContentType(data)
			if isImage(mimeType) {
				uploadResp, err = commandhandler.NewHandleSendImage(senderJidTypes, sliceJID, data, captionMsg)
			} else if isVideo(mimeType) {
				uploadResp, err = commandhandler.NewHandleSendVideo(senderJidTypes, sliceJID, data, captionMsg)
			} else if isAudio(mimeType) {
				uploadResp, err = commandhandler.NewHandleSendAudio(senderJidTypes, sliceJID, data)
			} else {
				uploadResp, err = commandhandler.NewHandleSendDocument(senderJidTypes, sliceJID, handler.Filename, data, captionMsg)
			}
			if err != nil {
				handleError(c.Writer, http.StatusInternalServerError, "Failed to handle file upload", err)
				return
			}

			resp = append(resp, uploadResp...)
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{"message": "gagal kirim, tolong hit endpoint untuk melakukan qrcode kembali"})
}

const (
	imageJPEG = "image/jpeg"
	imageJPG  = "image/jpg"
	imagePNG  = "image/png"
	imageWEBP = "image/webp"
	imageGIF  = "image/gif"
	imageAVIF = "image/avif"
	imageAPNG = "image/apng"
	imageSVG  = "image/svg+xml"

	//video
	videoMp4       = "video/mp4"
	videoMpeg      = "video/mpeg"
	videoOgg       = "video/ogg"
	videoWebm      = "video/webm"
	videoAvi       = "video/avi"
	videoQuickTime = "video/quicktime"
	videoWmv       = "video/x-ms-wmv"

	//audio
	audioMpeg = "audio/mpeg"
	audioOgg  = "audio/ogg"
	audioWav  = "audio/wav"
	audioWebm = "audio/webm"
	audioAac  = "audio/aac"
	audioMp4  = "audio/mp4"
)

func isImage(mimeType string) bool {
	extAsImage := []string{
		imageJPEG,
		imageJPG,
		imagePNG,
		imageWEBP,
		imageGIF,
		imageAVIF,
		imageAPNG,
		imageSVG,
	}
	return stringContains(extAsImage, mimeType)
}

func isVideo(mimeType string) bool {
	extAsImage := []string{
		videoMp4,
		videoMpeg,
		videoOgg,
		videoWebm,
		videoAvi,
		videoQuickTime,
		videoWmv,
	}
	return stringContains(extAsImage, mimeType)
}

func isAudio(mimeType string) bool {
	extAsImage := []string{
		audioMpeg,
		audioOgg,
		audioWav,
		audioWebm,
		audioAac,
		audioMp4,
	}
	return stringContains(extAsImage, mimeType)
}

func stringContains(strSlice []string, str string) bool {
	for _, val := range strSlice {
		if val == str {
			return true
		}
	}
	return false
}

func handleError(w http.ResponseWriter, statusCode int, message string, err error) {
	fmt.Errorf("%s: %v", message, err)
	http.Error(w, message, statusCode)
}

func (h Handler) HandleQR(c *gin.Context) {
	devices, err := h.CommandHandler.Container.GetAllDevices()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	fmt.Printf("list devices : %+v \n", devices)

	// Get query parameters
	senderString := c.Query("sender")
	if senderString == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "sender should be filled"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	if len(devices) > 0 {
		// Get specific QR code
		base64qrcode, err := h.CommandHandler.GetSpecificQR(context.Background(), commandhandler.Clients, senderJidTypes)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		if base64qrcode == "" {
			err = errors.New("you are already login")
			c.JSON(http.StatusOK, gin.H{"message": err.Error()})
			return
		}

		c.Data(http.StatusOK, "image/png", []byte(base64qrcode))
		return

	} else {
		// Get specific QR code
		base64qrcode, err := h.CommandHandler.GetSingleQR(context.Background(), commandhandler.Clients, senderJidTypes)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		if base64qrcode == "" {
			err = errors.New("you are already login")
			c.JSON(http.StatusOK, gin.H{"message": err.Error()})
			return
		}
		c.Data(http.StatusOK, "image/png", []byte(base64qrcode))
		return
	}
}

// Logout checks user status
func (h Handler) Logout(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusOK)
		return
	}

	// Get query parameters
	senderString := c.Query("sender")
	if senderString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "sender should be filled"})
		return
	}
	senderJidTypes := types.NewJID(senderString, types.DefaultUserServer)

	clientSpecificUser := commandhandler.Clients[senderJidTypes.User]
	if clientSpecificUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	err := clientSpecificUser.Logout()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success logout"})
}
