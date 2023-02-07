/*
Package webhooks provides a simple way to create a webhooks server in Go.

			Before using this API, You must set up/subscribe to a webhooks to receive notifications from the WhatsApp Business Platform.
			Follow Whatsapp Webhooks Getting Started guide (https://developers.facebook.com/docs/graph-api/webhooks/getting-started)
			to create your endpoint and configure your Webhooks. When you configure your Webhooks, make sure to choose WhatsApp Business
			Account and subscribe to one or more WhatsApp business account fields.

			Webhooks are triggered when a customer performs an action or the status for a message a business sends
			a customer changes.

			You get a webhooks notification, When a customer performs one of the following an action

			- Sends a text message to the business
			- Sends an image, video, audio, document, or sticker to the business
			- Sends contact information to the business
			- Sends location information to the business
			- Clicks a reply button set up by the business
			- Clicks a call-to-actions button on an Ad that Clicks to WhatsApp
			- Clicks an item on a business list
			- Updates their profile information such as their phone number
			- Asks for information about a specific product
			- Orders products being sold by the business

			Notification Payload Object
			NotificationPayloadObject is a combination of nested objects of JSON arrays and objects that contain information about a change.

			Structure of the notification payload object

			{
		  		"object": "whatsapp_business_account",
		  		"entry": [{
		    		"id": "WHATSAPP-BUSINESS-ACCOUNT-ID",
		    		"changes": [{
		      		"value": {
		         		"messaging_product": "whatsapp",
		         		"metadata": {
		           		"display_phone_number": "PHONE-NUMBER",
		           		"phone_number_id": "PHONE-NUMBER-ID"
		         		},
		      		# Additional arrays and objects
		         		"contacts": [{...}]
		         		"errors": [{...}]
		         		"messages": [{...}]
		         		"statuses": [{...}]
		      		},
		      		"field": "messages"
		    }]
		  }]
		}

		Example of a notification payload object after receiving a text message

	{
	  "object": "whatsapp_business_account",
	  "entry": [{
	      "id": "WHATSAPP_BUSINESS_ACCOUNT_ID",
	      "changes": [{
	          "value": {
	              "messaging_product": "whatsapp",
	              "metadata": {
	                  "display_phone_number": PHONE_NUMBER,
	                  "phone_number_id": PHONE_NUMBER_ID
	              },
	              "contacts": [{
	                  "profile": {
	                    "name": "NAME"
	                  },
	                  "wa_id": PHONE_NUMBER
	                }],
	              "messages": [{
	                  "from": PHONE_NUMBER,
	                  "id": "wamid.ID",
	                  "timestamp": TIMESTAMP,
	                  "text": {
	                    "body": "MESSAGE_BODY"
	                  },
	                  "type": "text"
	                }]
	          },
	          "field": "messages"
	        }]
	  }]
	}

	Message Status Updates
	he WhatsApp Business Platform sends notifications to inform you of the status of the messages between you and users.<\br>
	When a message is sent successfully, you receive a notification when the
	message is sent, delivered, and read. The order of these notifications in your app may not reflect
	the actual timing of the message status. View the timestamp to determine the timing, if necessary.
*/
package webhooks

import "github.com/piusalfred/whatsapp/pkg/models"

type (
	Metadata struct {
		DisplayPhoneNumber string `json:"display_phone_number,omitempty"`
		PhoneNumberID      string `json:"phone_number_id,omitempty"`
	}

	Profile struct {
		Name string `json:"name,omitempty"`
	}

	Contact struct {
		Profile *Profile `json:"profile,omitempty"`
		WaID    string   `json:"wa_id,omitempty"`
	}

	Message struct {
		From      string           `json:"from,omitempty"`
		ID        string           `json:"id,omitempty"`
		Timestamp int64            `json:"timestamp,omitempty"`
		Type      string           `json:"type,omitempty"`
		Errors    []*Error         `json:"errors,omitempty"`
		Text      *models.Text     `json:"text,omitempty"`
		Location  *models.Location `json:"location,omitempty"`
		Recation  *models.Reaction `json:"reaction,omitempty"`
		Sticker   *Media           `json:"sticker,omitempty"`
		Image     *Media           `json:"image,omitempty"`
		Video     *Media           `json:"video,omitempty"`
		Audio     *Media           `json:"audio,omitempty"`
		Document  *Media           `json:"document,omitempty"`
		Contacts  *models.Contacts `json:"contacts,omitempty"`
	}

	Value struct {
		MessagingProduct string     `json:"messaging_product,omitempty"`
		Metadata         *Metadata  `json:"metadata,omitempty"`
		Contacts         []*Contact `json:"contacts,omitempty"`
		Messages         []*Message `json:"messages,omitempty"`
	}

	Change struct {
		Value *Value `json:"value,omitempty"`
		Field string `json:"field,omitempty"`
	}

	Entry struct {
		ID      string    `json:"id,omitempty"`
		Changes []*Change `json:"changes,omitempty"`
	}

	Notification struct {
		Object string   `json:"object,omitempty"`
		Entry  []*Entry `json:"entry,omitempty"`
	}

	Error struct {
		Code    int    `json:"code,omitempty"`
		Details string `json:"details,omitempty"`
		Title   string `json:"title,omitempty"`
	}
)
