package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Ireoo/sixin-server/base"
)

// 响应结构体
type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// sendJSON 辅助函数，用于发送 JSON 响应
func sendJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

var messageHandler *base.MessageHandler

// SetMessageHandler 设置消息处理器
func SetMessageHandler(mh *base.MessageHandler) {
	messageHandler = mh
}

// Ping 处理 ping 请求
func Ping(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, Response{Message: "pong"})
}
