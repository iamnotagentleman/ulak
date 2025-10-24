package enums

type MessageSendingStatus string

const (
	StatusPending MessageSendingStatus = "PENDING"
	StatusSent    MessageSendingStatus = "SENT"
	StatusFailed  MessageSendingStatus = "FAILED"
)

type MessageChannel string

const (
	ChannelSMS     MessageChannel = "SMS"
	ChannelEmail   MessageChannel = "EMAIL"
	ChannelPush    MessageChannel = "PUSH"
	ChannelWebhook MessageChannel = "WEBHOOK"
)

type AutoSendAction string

const (
	AutoSendStart AutoSendAction = "start"
	AutoSendStop  AutoSendAction = "stop"
)
