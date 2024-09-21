package room

import (
	"net/http"

	"github.com/Ireoo/sixin-server/models"
	"gorm.io/gorm"
)

type RoomHandler struct {
	DB *gorm.DB
}

func NewRoomHandler(db *gorm.DB) *RoomHandler {
	return &RoomHandler{DB: db}
}

func (rh *RoomHandler) GetRooms(w http.ResponseWriter, r *http.Request) {
	var rooms []models.Room
	if err := rh.DB.Preload("Owner").Preload("Members").Find(&rooms).Error; err != nil {
		http.Error(w, "获取房间列表失败", http.StatusInternalServerError)
		return
	}
	// 返回房间列表
}

func (rh *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var room models.Room
	// 解析请求体并创建房间
	if err := rh.DB.Create(&room).Error; err != nil {
		http.Error(w, "创建房间失败", http.StatusInternalServerError)
		return
	}
	// 返回创建的房间
}

func (rh *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request, id string) {
	var room models.Room
	if err := rh.DB.Preload("Owner").Preload("Members").First(&room, id).Error; err != nil {
		http.Error(w, "获取房间失败", http.StatusNotFound)
		return
	}
	// 返回房间信息
}

func (rh *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request, id string) {
	var room models.Room
	if err := rh.DB.First(&room, id).Error; err != nil {
		http.Error(w, "房间不存在", http.StatusNotFound)
		return
	}
	// 更新房间信息
	// 返回更新后的房间信息
}

func (rh *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request, id string) {
	if err := rh.DB.Delete(&models.Room{}, id).Error; err != nil {
		http.Error(w, "删除房间失败", http.StatusInternalServerError)
		return
	}
	// 返回删除成功的消息
}

func (rh *RoomHandler) GetRoomByUsers(w http.ResponseWriter, r *http.Request) {
	// 实现获取用户房间的逻辑
}
