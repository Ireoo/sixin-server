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

	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/logger"
	"github.com/gorilla/websocket"
	"github.com/zishang520/socket.io/v2/socket"
	"gopkg.in/gomail.v2"
)

type Base struct {
	Folder        string
	Self          map[string]interface{}
	Qrcode        string
	TargetName    []string
	ChatlogsName  []string
	EmailNote     bool
	ZhuanfaGroup  []string
	Messages      map[string][]string
	Sendme        bool
	Interval      *time.Ticker
	ReceiveDevice bool
	Config        map[string]interface{}
	mu            sync.Mutex
	IoManager     *socket.Server
	WsManager     []*websocket.Conn
	DbManager     *database.DatabaseManager
}

func NewBase(cfg *config.Config) *Base {
	b := &Base{}

	// 创建 DatabaseManager 实例
	dbManager, err := database.NewDatabaseManager(database.DatabaseType(cfg.DBType), cfg.DBConn)
	if err != nil {
		logger.Error("创建数据库管理器失败:", err)
		return nil
	}

	// 将数据库实例和管理器保存到 base 中
	b.DbManager = dbManager

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

	if text == "wechat:receive" || text == "wechat:message" {
		return
	}

	// 添加消息到历史记录
	mh.Messages["m5stack"] = append(mh.Messages["m5stack"], text)
	mh.Messages["telegram"] = append(mh.Messages["telegram"], msg)

	if len(mh.Messages["m5stack"]) > 10 {
		mh.Messages["m5stack"] = mh.Messages["m5stack"][1:]
		mh.Messages["telegram"] = mh.Messages["telegram"][1:]
	}

	// 发送消息给 Socket.IO 客户端
	if mh.IoManager != nil {
		mh.IoManager.Emit("message", map[string]string{"text": text, "msg": msg})
	}

	// 发送消息给 WebSocket 客户端
	if mh.EmailNote {
		mh.SendEmail()
	}

	// 保存消息到文件
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

func (b *Base) SetIO(io *socket.Server) {
	b.IoManager = io
}

func (b *Base) AddWs(ws *websocket.Conn) {
	b.WsManager = append(b.WsManager, ws)
}

func (b *Base) RemoveWs(ws *websocket.Conn) {
	for i, v := range b.WsManager {
		if v == ws {
			b.WsManager = append(b.WsManager[:i], b.WsManager[i+1:]...)

			break
		}
	}
}

func (b *Base) SetDatabaseManager(dbManager *database.DatabaseManager) {

	b.DbManager = dbManager
}

func (b *Base) GetWs() []*websocket.Conn {
	return b.WsManager
}

// 添加 sendMessageToUsers 方法
func (b *Base) SendMessageToUsers(message interface{}, userIDs ...uint) {
	for _, userID := range userIDs {
		socketID := socket.SocketId(fmt.Sprintf("%d", userID))
		clients := b.IoManager.Sockets().Sockets()
		clients.Range(func(id socket.SocketId, client *socket.Socket) bool {
			if client.Id() == socketID {
				err := client.Emit("message", message)
				if err != nil {
					fmt.Printf("发送消息给用户 %d 失败: %v\n", userID, err)
				}
				return false
			}
			return true
		})
	}
}
