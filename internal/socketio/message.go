package socketio

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Ireoo/sixin-server/models"
	"github.com/go-playground/validator/v10"
	"github.com/patrickmn/go-cache"
	"github.com/zishang520/socket.io/v2/socket"
)

var validate = validator.New()
var messagePool = sync.Pool{
	New: func() interface{} {
		return &models.Message{}
	},
}

func (sim *SocketIOManager) handleGetChats(client *socket.Socket, args ...any) {
	userID, err := sim.getUserIDOrEmitError(client)
	if err != nil {
		return
	}

	cacheKey := fmt.Sprintf("userChats_%d", userID)
	if cachedChats, found := sim.cache.Get(cacheKey); found {
		client.Emit("getChats", cachedChats)
		return
	}

	messages, err := sim.baseInstance.DbManager.GetChats(userID)
	if err != nil {
		emitErrorAndLog(client, "获取聊天记录失败", err)
		return
	}

	sim.cache.Set(cacheKey, messages, cache.DefaultExpiration)
	client.Emit("getChats", messages)
}

func (sim *SocketIOManager) handleMessage(client *socket.Socket, args ...any) {
	msgBytes, err := checkArgsAndType[[]byte](args, 0)
	if err != nil {
		emitErrorAndLog(client, "缺少消息内容或消息格式错误", err)
		return
	}

	fmt.Println("收到消息：", string(msgBytes))

	message := messagePool.Get().(*models.Message)
	defer messagePool.Put(message)

	if err := json.Unmarshal(msgBytes, message); err != nil {
		emitErrorAndLog(client, "解析消息失败", err)
		return
	}

	if err := validate.Struct(message); err != nil {
		emitErrorAndLog(client, "消息内容不合法", err)
		return
	}

	userID, err := sim.getUserIDOrEmitError(client)
	if err != nil {
		return
	}
	message.TalkerID = userID

	// 创建一个带有超时的 context，用于控制 goroutine 执行时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("消息保存操作超时")
				emitError(client, "保存消息超时", ctx.Err())
			}
			return
		default:
			if err := sim.baseInstance.DbManager.CreateMessage(message); err != nil {
				emitErrorAndLog(client, "保存消息失败", err)
				return
			}

			fullMessage := *message

			recipientID := message.ListenerID
			if recipientID == 0 {
				recipientID = message.RoomID
			}

			sim.SendMessageToUsers(fullMessage, message.TalkerID, recipientID)
		}
	}(ctx)
}

func (sim *SocketIOManager) getUserIDOrEmitError(client *socket.Socket) (uint, error) {
	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitErrorAndLog(client, "userID 类型转换失败", err)
		return 0, err
	}
	return userID, nil
}

func emitErrorAndLog(client *socket.Socket, message string, err error) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	}
	emitError(client, message, err)
}
