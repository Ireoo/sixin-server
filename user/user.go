package user

import (
	"net/http"

	"github.com/Ireoo/sixin-server/models"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

func (uh *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := uh.DB.Find(&users).Error; err != nil {
		http.Error(w, "获取用户列表失败", http.StatusInternalServerError)
		return
	}
	// 返回用户列表
}

func (uh *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	// 解析请求体并创建用户
	if err := uh.DB.Create(&user).Error; err != nil {
		http.Error(w, "创建用户失败", http.StatusInternalServerError)
		return
	}
	// 返回创建的用户
}

func (uh *UserHandler) GetUser(w http.ResponseWriter, r *http.Request, id string) {
	var user models.User
	if err := uh.DB.First(&user, id).Error; err != nil {
		http.Error(w, "获取用户失败", http.StatusNotFound)
		return
	}
	// 返回用户信息
}

func (uh *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	var user models.User
	if err := uh.DB.First(&user, id).Error; err != nil {
		http.Error(w, "用户不存在", http.StatusNotFound)
		return
	}
	// 更新用户信息
	// 返回更新后的用户信息
}

func (uh *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if err := uh.DB.Delete(&models.User{}, id).Error; err != nil {
		http.Error(w, "删除用户失败", http.StatusInternalServerError)
		return
	}
	// 返回删除成功的消息
}
