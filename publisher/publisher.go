package publisher

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramPublisher struct {
	ChannelID string // Telegram channel id (e.g. @my_channel)
	BotAPI    *tgbotapi.BotAPI
}

func NewTelegramPublisher(channelId, token string) (*TelegramPublisher, error) {
	b, e := tgbotapi.NewBotAPI(token)
	if e != nil {
		return nil, e
	}
	return &TelegramPublisher{
		ChannelID: channelId,
		BotAPI:    b,
	}, nil
}

func (t *TelegramPublisher) Publish(msg string) (pubID string, err error) {
	tgMsg := tgbotapi.NewMessageToChannel(t.ChannelID, msg)

	s, err := t.BotAPI.Send(tgMsg)
	if err != nil {
		return "", err
	}
	return string(rune(s.MessageID)), nil
}
