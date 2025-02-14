/*
 * Copyright 2023 Pius Alfred <me.pius1102@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software
 * and associated documentation files (the “Software”), to deal in the Software without restriction,
 * including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or substantial
 * portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT
 * LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
 * WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	whttp "github.com/piusalfred/whatsapp/http"
	"github.com/piusalfred/whatsapp/models"
	"github.com/piusalfred/whatsapp/qrcodes"
)

var ErrNilRequest = errors.New("nil request")

const (
	BaseURL                   = "https://graph.facebook.com/"
	LowestSupportedVersion    = "v16.0"
	ContactBirthDayDateFormat = "2006-01-02" // YYYY-MM-DD
)

const (
	TextMessageType        = "text"
	ReactionMessageType    = "reaction"
	MediaMessageType       = "media"
	LocationMessageType    = "location"
	ContactMessageType     = "contact"
	InteractiveMessageType = "interactive"
)

const (
	MaxAudioSize         = 16 * 1024 * 1024  // 16 MB
	MaxDocSize           = 100 * 1024 * 1024 // 100 MB
	MaxImageSize         = 5 * 1024 * 1024   // 5 MB
	MaxVideoSize         = 16 * 1024 * 1024  // 16 MB
	MaxStickerSize       = 100 * 1024        // 100 KB
	UploadedMediaTTL     = 30 * 24 * time.Hour
	MediaDownloadLinkTTL = 5 * time.Minute
)

const (
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
	MediaTypeImage    MediaType = "image"
	MediaTypeSticker  MediaType = "sticker"
	MediaTypeVideo    MediaType = "video"
)

// MediaMaxAllowedSize returns the allowed maximum size for media. It returns
// -1 for unknown media type. Currently, it checks for MediaTypeAudio,MediaTypeVideo,
// MediaTypeImage, MediaTypeSticker,MediaTypeDocument.
func MediaMaxAllowedSize(mediaType MediaType) int {
	sizeMap := map[MediaType]int{
		MediaTypeAudio:    MaxAudioSize,
		MediaTypeDocument: MaxDocSize,
		MediaTypeSticker:  MaxStickerSize,
		MediaTypeImage:    MaxImageSize,
		MediaTypeVideo:    MaxVideoSize,
	}

	size, ok := sizeMap[mediaType]
	if ok {
		return size
	}

	return -1
}

type (
	ResponseMessage struct {
		Product  string             `json:"messaging_product,omitempty"`
		Contacts []*ResponseContact `json:"contacts,omitempty"`
		Messages []*MessageID       `json:"messages,omitempty"`
	}
	MessageID struct {
		ID string `json:"id,omitempty"`
	}

	ResponseContact struct {
		Input      string `json:"input"`
		WhatsappID string `json:"wa_id"`
	}

	// MessageType represents the type of message currently supported.
	// Which are Text messages,Reaction messages,Media messages,Location messages,Contact messages,
	// and Interactive messages.
	// You may also send any of these message types as a reply, except reaction messages.
	// For more go to https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages
	MessageType string

	// Client includes the http client, base url, apiVersion, access token, phone number id,
	// and whatsapp business account id.
	// which are used to make requests to the whatsapp api.
	// Example:
	// 	client := whatsapp.NewClient(
	// 		whatsapp.WithHTTPClient(http.DefaultClient),
	// 		whatsapp.WithBaseURL(whatsapp.BaseURL),
	// 		whatsapp.WithVersion(whatsapp.LowestSupportedVersion),
	// 		whatsapp.WithAccessToken("access_token"),
	// 		whatsapp.WithPhoneNumberID("phone_number_id"),
	// 		whatsapp.WithBusinessAccountID("whatsapp_business_account_id"),
	// 	)
	//  // create a text message
	//  message := whatsapp.TextMessage{
	//  	Recipient: "<phone_number>",
	//  	Message:   "Hello World",
	//      PreviewURL: false,
	//  }
	// // send the text message
	//  _, err := client.SendTextMessage(context.Background(), message)
	//  if err != nil {
	//  	log.Fatal(err)
	//  }
	Client struct {
		rwm               *sync.RWMutex
		http              *http.Client
		baseURL           string
		apiVersion        string
		accessToken       string
		phoneNumberID     string
		businessAccountID string
	}

	ClientOption func(*Client)
)

func WithHTTPClient(http *http.Client) ClientOption {
	return func(client *Client) {
		client.http = http
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) {
		client.baseURL = baseURL
	}
}

func WithVersion(version string) ClientOption {
	return func(client *Client) {
		client.apiVersion = version
	}
}

func WithAccessToken(accessToken string) ClientOption {
	return func(client *Client) {
		client.accessToken = accessToken
	}
}

func WithPhoneNumberID(phoneNumberID string) ClientOption {
	return func(client *Client) {
		client.phoneNumberID = phoneNumberID
	}
}

func WithBusinessAccountID(whatsappBusinessAccountID string) ClientOption {
	return func(client *Client) {
		client.businessAccountID = whatsappBusinessAccountID
	}
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		rwm:               &sync.RWMutex{},
		http:              http.DefaultClient,
		baseURL:           BaseURL,
		apiVersion:        "v16.0",
		accessToken:       "",
		phoneNumberID:     "",
		businessAccountID: "",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

type clientContext struct {
	baseURL           string
	apiVersion        string
	accessToken       string
	phoneNumberID     string
	businessAccountID string
}

func (client *Client) context() *clientContext {
	client.rwm.RLock()
	defer client.rwm.RUnlock()

	return &clientContext{
		baseURL:           client.baseURL,
		apiVersion:        client.apiVersion,
		accessToken:       client.accessToken,
		phoneNumberID:     client.phoneNumberID,
		businessAccountID: client.businessAccountID,
	}
}

func (client *Client) SetAccessToken(accessToken string) {
	client.rwm.Lock()
	defer client.rwm.Unlock()
	client.accessToken = accessToken
}

func (client *Client) SetPhoneNumberID(phoneNumberID string) {
	client.rwm.Lock()
	defer client.rwm.Unlock()
	client.phoneNumberID = phoneNumberID
}

func (client *Client) SetBusinessAccountID(businessAccountID string) {
	client.rwm.Lock()
	defer client.rwm.Unlock()
	client.businessAccountID = businessAccountID
}

type TextMessage struct {
	Message    string
	PreviewURL bool
}

// SendTextMessage sends a text message to a WhatsApp Business Account.
func (client *Client) SendTextMessage(ctx context.Context, recipient string,
	message *TextMessage,
) (*ResponseMessage, error) {
	cctx := client.context()
	request := &SendTextRequest{
		BaseURL:       cctx.baseURL,
		AccessToken:   cctx.accessToken,
		PhoneNumberID: cctx.phoneNumberID,
		ApiVersion:    cctx.apiVersion,
		Recipient:     recipient,
		Message:       message.Message,
		PreviewURL:    message.PreviewURL,
	}
	resp, err := SendText(ctx, client.http, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send text message: %w", err)
	}

	return resp, nil
}

// SendLocationMessage sends a location message to a WhatsApp Business Account.
func (client *Client) SendLocationMessage(ctx context.Context, recipient string,
	message *models.Location,
) (*ResponseMessage, error) {
	request := &SendLocationRequest{
		BaseURL:       client.baseURL,
		AccessToken:   client.accessToken,
		PhoneNumberID: client.phoneNumberID,
		ApiVersion:    client.apiVersion,
		Recipient:     recipient,
		Name:          message.Name,
		Address:       message.Address,
		Latitude:      message.Latitude,
		Longitude:     message.Longitude,
	}

	resp, err := SendLocation(ctx, client.http, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send location message: %w", err)
	}

	return resp, nil
}

type ReactMessage struct {
	MessageID string
	Emoji     string
}

func (client *Client) React(ctx context.Context, recipient string, req *ReactMessage) (*ResponseMessage, error) {
	cctx := client.context()
	request := &ReactRequest{
		BaseURL:       cctx.baseURL,
		AccessToken:   cctx.accessToken,
		PhoneNumberID: cctx.phoneNumberID,
		ApiVersion:    cctx.apiVersion,
		Recipient:     recipient,
		MessageID:     req.MessageID,
		Emoji:         req.Emoji,
	}

	resp, err := React(ctx, client.http, request)
	if err != nil {
		return nil, fmt.Errorf("react: %w", err)
	}

	return resp, nil
}

type MediaMessage struct {
	Type      MediaType
	MediaID   string
	MediaLink string
	Caption   string
	Filename  string
	Provider  string
}

// SendMedia sends a media message to the recipient.
func (client *Client) SendMedia(ctx context.Context, recipient string, req *MediaMessage,
	cacheOptions *CacheOptions,
) (*ResponseMessage, error) {
	cctx := client.context()
	request := &SendMediaRequest{
		BaseURL:       cctx.baseURL,
		AccessToken:   cctx.accessToken,
		PhoneNumberID: cctx.phoneNumberID,
		ApiVersion:    cctx.apiVersion,
		Recipient:     recipient,
		Type:          req.Type,
		MediaID:       req.MediaID,
		MediaLink:     req.MediaLink,
		Caption:       req.Caption,
		Filename:      req.Filename,
		Provider:      req.Provider,
		CacheOptions:  cacheOptions,
	}

	resp, err := SendMedia(ctx, client.http, request)
	if err != nil {
		return nil, fmt.Errorf("client send media: %w", err)
	}

	return resp, nil
}

// ReplyMessage is a message that is sent as a reply to a previous message. The previous message's ID
// is needed and is set as Context in ReplyRequest.
// Content is the message content. It can be a Text, Location, Media, Template, or Contact.
type ReplyMessage struct {
	Context string
	Type    MessageType
	Content any
}

func (client *Client) Reply(ctx context.Context, recipient string, req *ReplyMessage) (*ResponseMessage, error) {
	cctx := client.context()
	request := &ReplyRequest{
		BaseURL:       cctx.baseURL,
		AccessToken:   cctx.accessToken,
		PhoneNumberID: cctx.phoneNumberID,
		ApiVersion:    cctx.apiVersion,
		Recipient:     recipient,
		Context:       req.Context,
		MessageType:   req.Type,
		Content:       req.Content,
	}

	resp, err := Reply(ctx, client.http, request)
	if err != nil {
		return nil, fmt.Errorf("client reply: %w", err)
	}

	return resp, nil
}

func (client *Client) SendContacts(ctx context.Context, recipient string, contacts *models.Contacts) (
	*ResponseMessage, error,
) {
	cctx := client.context()
	req := &SendContactRequest{
		BaseURL:       cctx.baseURL,
		AccessToken:   cctx.accessToken,
		PhoneNumberID: cctx.phoneNumberID,
		ApiVersion:    cctx.apiVersion,
		Recipient:     recipient,
		Contacts:      contacts,
	}

	resp, err := SendContact(ctx, client.http, req)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

// MarkMessageRead sends a read receipt for a message.
func (client *Client) MarkMessageRead(ctx context.Context, messageID string) (*StatusResponse, error) {
	reqBody := &MessageStatusUpdateRequest{
		MessagingProduct: "whatsapp",
		Status:           MessageStatusRead,
		MessageID:        messageID,
	}

	cctx := client.context()

	reqCtx := &whttp.RequestContext{
		Name:       "mark read",
		BaseURL:    cctx.baseURL,
		ApiVersion: cctx.apiVersion,
		SenderID:   cctx.phoneNumberID,
		Endpoints:  []string{"/messages"},
	}

	params := &whttp.Request{
		Context: reqCtx,
		Method:  http.MethodPost,
		Headers: map[string]string{"Content-Type": "application/json"},
		Bearer:  cctx.accessToken,
		Payload: reqBody,
	}

	var success StatusResponse
	err := whttp.Send(ctx, client.http, params, &success)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return &success, nil
}

type Template struct {
	LanguageCode   string
	LanguagePolicy string
	Name           string
	Components     []*models.TemplateComponent
}

// SendTemplate sends a template message to the recipient.
func (client *Client) SendTemplate(ctx context.Context, recipient string, req *Template) (*ResponseMessage, error) {
	cctx := client.context()
	request := &SendTemplateRequest{
		BaseURL:                cctx.baseURL,
		AccessToken:            cctx.accessToken,
		PhoneNumberID:          cctx.phoneNumberID,
		ApiVersion:             cctx.apiVersion,
		Recipient:              recipient,
		TemplateLanguageCode:   req.LanguageCode,
		TemplateLanguagePolicy: req.LanguagePolicy,
		TemplateName:           req.Name,
		TemplateComponents:     req.Components,
	}

	resp, err := SendTemplate(ctx, client.http, request)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

////////////// QrCode

func (client *Client) CreateQrCode(ctx context.Context, message *qrcodes.CreateRequest) (
	*qrcodes.CreateResponse, error,
) {
	request := &qrcodes.CreateRequest{
		PrefilledMessage: message.PrefilledMessage,
		ImageFormat:      message.ImageFormat,
	}

	cctx := client.context()

	rctx := &qrcodes.RequestContext{
		BaseURL:     cctx.baseURL,
		PhoneID:     cctx.phoneNumberID,
		ApiVersion:  cctx.apiVersion,
		AccessToken: client.accessToken,
	}
	resp, err := qrcodes.Create(ctx, client.http, rctx, request)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

func (client *Client) ListQrCodes(ctx context.Context) (*qrcodes.ListResponse, error) {
	cctx := client.context()
	rctx := &qrcodes.RequestContext{
		BaseURL:     cctx.baseURL,
		PhoneID:     cctx.phoneNumberID,
		ApiVersion:  cctx.apiVersion,
		AccessToken: cctx.accessToken,
	}

	resp, err := qrcodes.List(ctx, client.http, rctx)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

func (client *Client) GetQrCode(ctx context.Context, qrCodeID string) (*qrcodes.Information, error) {
	cctx := client.context()
	rctx := &qrcodes.RequestContext{
		BaseURL:     cctx.baseURL,
		PhoneID:     cctx.phoneNumberID,
		ApiVersion:  cctx.apiVersion,
		AccessToken: cctx.accessToken,
	}

	resp, err := qrcodes.Get(ctx, client.http, rctx, qrCodeID)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

func (client *Client) UpdateQrCode(ctx context.Context, qrCodeID string, request *qrcodes.CreateRequest,
) (*qrcodes.SuccessResponse, error) {
	cctx := client.context()
	rctx := &qrcodes.RequestContext{
		BaseURL:     cctx.baseURL,
		PhoneID:     cctx.phoneNumberID,
		ApiVersion:  cctx.apiVersion,
		AccessToken: cctx.accessToken,
	}

	resp, err := qrcodes.Update(ctx, client.http, rctx, qrCodeID, request)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

func (client *Client) DeleteQrCode(ctx context.Context, qrCodeID string) (*qrcodes.SuccessResponse, error) {
	cctx := client.context()
	rctx := &qrcodes.RequestContext{
		BaseURL:     cctx.baseURL,
		PhoneID:     cctx.phoneNumberID,
		ApiVersion:  cctx.apiVersion,
		AccessToken: cctx.accessToken,
	}

	resp, err := qrcodes.Delete(ctx, client.http, rctx, qrCodeID)
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}

	return resp, nil
}

////// PHONE NUMBERS

func (client *Client) RequestVerificationCode(ctx context.Context, codeMethod string, language string) error {
	cctx := client.context()
	if err := RequestCode(ctx, client.http, &VerificationCodeRequest{
		Token:         cctx.accessToken,
		BaseURL:       cctx.baseURL,
		ApiVersion:    cctx.apiVersion,
		PhoneNumberID: cctx.phoneNumberID,
		CodeMethod:    codeMethod,
		Language:      language,
	}); err != nil {
		return fmt.Errorf("client: %w", err)
	}

	return nil
}
