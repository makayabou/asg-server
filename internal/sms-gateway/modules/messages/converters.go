package messages

import (
	"fmt"
	"math"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/capcom6/go-helpers/slices"
)

func messageToDomain(input Message) (MessageOut, error) {
	var ttl *uint64 = nil
	if input.ValidUntil != nil {
		secondsUntil := uint64(math.Max(0, time.Until(*input.ValidUntil).Seconds()))
		ttl = &secondsUntil
	}

	textContent, err := input.GetTextContent()
	if err != nil {
		return MessageOut{}, fmt.Errorf("can't get text content: %w", err)
	}
	dataContent, err := input.GetDataContent()
	if err != nil {
		return MessageOut{}, fmt.Errorf("can't get data content: %w", err)
	}

	out := MessageOut{
		MessageIn: MessageIn{
			ID: input.ExtID,

			TextContent: textContent,
			DataContent: dataContent,

			PhoneNumbers:       slices.Map(input.Recipients, recipientToDomain),
			IsEncrypted:        input.IsEncrypted,
			SimNumber:          input.SimNumber,
			WithDeliveryReport: &input.WithDeliveryReport,
			TTL:                ttl,
			ValidUntil:         input.ValidUntil,
			Priority:           smsgateway.MessagePriority(input.Priority),
		},
		CreatedAt: input.CreatedAt,
	}
	if len(input.States) > 0 || input.DeviceID != "" {
        state := modelToMessageState(input)
        out.State = &state
    }
	return out, nil
}

func recipientToDomain(input MessageRecipient) string {
	return input.PhoneNumber
}
