package input

type Message struct {
	ClientMessageID string `json:"client_message_id" binding:"required"`
	Title           string `json:"title" binding:"required"`
	Content         string `json:"content" binding:"required"`
}
