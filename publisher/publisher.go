package publisher

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strconv"
)

type TelegramPublisher struct {
	ChannelID string // Telegram channel id (e.g. @my_channel)
	BotAPI    *tgbotapi.BotAPI
}

func NewTelegramPublisher(channelID, token string) (*TelegramPublisher, error) {
	b, e := tgbotapi.NewBotAPI(token)
	if e != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", e)
	}
	return &TelegramPublisher{
		ChannelID: channelID,
		BotAPI:    b,
	}, nil
}

func (t *TelegramPublisher) Publish(msg string) (pubID string, err error) {
	tgMsg := tgbotapi.NewMessageToChannel(t.ChannelID, msg)
	tgMsg.ParseMode = tgbotapi.ModeMarkdown
	tgMsg.DisableWebPagePreview = true

	m, err := t.BotAPI.Send(tgMsg)
	if err != nil {
		return "", fmt.Errorf("failed to send message to Telegram: %w", err)
	}
	return strconv.Itoa(m.MessageID), nil
}
