package publisher

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"strconv"
)

type TelegramPublisher struct {
	ChannelID     string // Telegram channel id (e.g. @my_channel)
	BotAPI        *tgbotapi.BotAPI
	ShouldPublish bool // If false, will print the message to the console (for development)
}

func NewTelegramPublisher(channelID string, token string, shouldPublish bool) (*TelegramPublisher, error) {
	b, e := tgbotapi.NewBotAPI(token)
	if e != nil {
		return nil, errlvl.Wrap(fmt.Errorf("failed to create Telegram bot: %w", e), errlvl.ERROR)
	}
	return &TelegramPublisher{
		ChannelID:     channelID,
		BotAPI:        b,
		ShouldPublish: shouldPublish,
	}, nil
}

func (t *TelegramPublisher) Publish(msg string) (pubID string, err error) {
	if !t.ShouldPublish {
		fmt.Println(msg)
		return "", nil
	}

	tgMsg := tgbotapi.NewMessageToChannel(t.ChannelID, msg)
	tgMsg.ParseMode = tgbotapi.ModeMarkdown
	tgMsg.DisableWebPagePreview = true

	m, err := t.BotAPI.Send(tgMsg)
	if err != nil {
		return "", errlvl.Wrap(fmt.Errorf("failed to send message to Telegram: %w", err), errlvl.ERROR)
	}
	return strconv.Itoa(m.MessageID), nil
}
