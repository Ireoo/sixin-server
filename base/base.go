package base

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zishang520/socket.io/v2/socket"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type Base struct {
	Folder        string
	Self          map[string]interface{}
	Qrcode        string
	TargetName    []string
	ChatlogsName  []string
	EmailNote     bool
	ZhuanfaGroup  []string
	WsGroup       map[string][]*websocket.Conn
	Messages      map[string][]string
	Sendme        bool
	Interval      *time.Ticker
	ReceiveDevice bool
	Config        map[string]interface{}
	DB            *gorm.DB
	mu            sync.Mutex
	wsConnMu      sync.RWMutex
	IO            *socket.Server
}

func NewMessageHandler(db *gorm.DB) *Base {
	mh := &Base{
		Folder:       "./data",
		Self:         make(map[string]interface{}),
		TargetName:   []string{"香蕉内个布呐呐", "强制分享 cium"},
		ChatlogsName: []string{"香蕉内个布呐呐", "王超", "L."},
		EmailNote:    false,
		ZhuanfaGroup: []string{},
		WsGroup: map[string][]*websocket.Conn{
			"m5stack":  {},
			"telegram": {},
			"web":      {},
		},
		Messages: map[string][]string{
			"m5stack":  {},
			"telegram": {},
		},
		Sendme:        true,
		ReceiveDevice: true,
		Config:        make(map[string]interface{}),
		DB:            db,
	}

	mh.loadConfig()
	mh.createSubfolders()
	mh.initMessages()

	return mh
}

func NewBase() *Base {
	b := &Base{
		Folder:       "./data",
		Self:         make(map[string]interface{}),
		TargetName:   []string{"香蕉内个布呐呐", "强制分享 cium"},
		ChatlogsName: []string{"香蕉内个布呐呐", "王超", "L."},
		EmailNote:    false,
		ZhuanfaGroup: []string{},
		WsGroup: map[string][]*websocket.Conn{
			"m5stack":  {},
			"telegram": {},
			"web":      {},
		},
		Messages: map[string][]string{
			"m5stack":  {},
			"telegram": {},
		},
		Sendme:        true,
		ReceiveDevice: true,
		Config:        make(map[string]interface{}),
		IO:            nil, // 初始化为 nil,稍后在 SetIO 方法中设置
	}

	b.loadConfig()
	b.createSubfolders()
	b.initMessages()

	return b
}

func (mh *Base) loadConfig() {
	configPath := filepath.Join(".", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Error reading config file: %v\n", err)
			return
		}
		if err := json.Unmarshal(data, &mh.Config); err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
			return
		}
		for k, v := range mh.Config {
			mh.set(k, v)
		}
	}
}

func (mh *Base) createSubfolders() {
	subfolders := []string{"image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, subfolder := range subfolders {
		path := filepath.Join(mh.Folder, subfolder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			fmt.Printf("Error creating folder %s: %v\n", path, err)
		}
	}
}

func (mh *Base) saveConfig() {
	configPath := filepath.Join(".", "config.json")
	data, err := json.MarshalIndent(mh.Config, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		return
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
	}
}

func (mh *Base) initMessages() {
	messagesPath := "messages.json"
	if _, err := os.Stat(messagesPath); err == nil {
		data, err := os.ReadFile(messagesPath)
		if err != nil {
			fmt.Printf("Error reading messages file: %v\n", err)
			return
		}
		if err := json.Unmarshal(data, &mh.Messages); err != nil {
			fmt.Printf("Error parsing messages file: %v\n", err)
			return
		}
		fmt.Printf("Read %d messages.\n", len(mh.Messages["m5stack"]))
	}
}

func (mh *Base) DownloadFile(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (mh *Base) GenerateVerificationCode() string {
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000)
}

func (mh *Base) SendMessage(text, msg string) {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	for _, ws := range mh.WsGroup["m5stack"] {
		if err := ws.WriteMessage(websocket.TextMessage, []byte(text)); err == nil {
			fmt.Printf("[m5stack] Sent message: %s\n", text)
		}
	}

	for _, ws := range mh.WsGroup["telegram"] {
		if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err == nil {
			fmt.Printf("[telegram] Sent message: %s\n", msg)
		}
	}

	if text == "wechat:receive" || text == "wechat:message" {
		return
	}

	mh.Messages["m5stack"] = append(mh.Messages["m5stack"], text)
	mh.Messages["telegram"] = append(mh.Messages["telegram"], msg)

	if len(mh.Messages["m5stack"]) > 10 {
		mh.Messages["m5stack"] = mh.Messages["m5stack"][1:]
		mh.Messages["telegram"] = mh.Messages["telegram"][1:]
	}

	if mh.EmailNote {
		mh.SendEmail()
	}

	data, err := json.MarshalIndent(mh.Messages, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling messages: %v\n", err)
		return
	}
	if err := os.WriteFile("messages.json", data, 0644); err != nil {
		fmt.Printf("Error writing messages file: %v\n", err)
	}
}

func (mh *Base) SendEmail() {
	m := gomail.NewMessage()
	m.SetHeader("From", "2636466208@qq.com")
	m.SetHeader("To", "19980108@qq.com")
	m.SetHeader("Subject", "Verification code for registering integem.com")
	verificationCode := mh.GenerateVerificationCode()
	m.SetBody("text/html", fmt.Sprintf("The verification code you obtained is %s", verificationCode))

	d := gomail.NewDialer("smtp.qq.com", 587, "2636466208@qq.com", "vvupgrgxbpoaeajj")

	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("Error sending email: %v\n", err)
	} else {
		fmt.Println("Email sent successfully")
	}
}

func (mh *Base) SaveChatlogs(name, msg string) {
	logEntry := fmt.Sprintf(`{"role":"%s","content":"%s","time":%d},`, name, msg, time.Now().UnixNano()/int64(time.Millisecond))
	if err := os.WriteFile("chats.txt", []byte(logEntry), os.ModeAppend); err != nil {
		fmt.Printf("Error writing chat logs: %v\n", err)
	}
}

func (mh *Base) set(key string, value interface{}) {
	switch key {
	case "Self":
		mh.Self = value.(map[string]interface{})
	case "Qrcode":
		mh.Qrcode = value.(string)
	case "EmailNote":
		mh.EmailNote = value.(bool)
	case "ReceiveDevice":
		mh.ReceiveDevice = value.(bool)
	}
	mh.Config[key] = value
	mh.saveConfig()
}

func (b *Base) SetDB(db *gorm.DB) {
	b.DB = db
}

func (b *Base) AddWebSocketConn(connType string, conn *websocket.Conn) {
	b.wsConnMu.Lock()
	defer b.wsConnMu.Unlock()
	b.WsGroup[connType] = append(b.WsGroup[connType], conn)
}

func (b *Base) RemoveWebSocketConn(connType string, conn *websocket.Conn) {
	b.wsConnMu.Lock()
	defer b.wsConnMu.Unlock()
	for i, c := range b.WsGroup[connType] {
		if c == conn {
			b.WsGroup[connType] = append(b.WsGroup[connType][:i], b.WsGroup[connType][i+1:]...)
			break
		}
	}
}

func (b *Base) HandleWebSocketMessage(connType string, messageType int, message []byte) {
	// 根据连接类型和消息类型处理消息
	switch connType {
	case "m5stack":
		// 处理 m5stack 消息
	case "telegram":
		// 处理 telegram 消息
	case "web":
		// 处理 web 消息
	}
	// 可以根据需要调用 SendMessage 方法
	b.SendMessage(string(message), string(message))
}

func (b *Base) SetIO(io *socket.Server) {
	b.IO = io
}
