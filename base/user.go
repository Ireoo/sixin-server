package base

import (
	"github.com/Ireoo/sixin-server/models"
)

type UserHandler struct {
	Base *Base
}

func NewUserHandler(base *Base) *UserHandler {
	return &UserHandler{Base: base}
}

func (uh *UserHandler) GetUsers() ([]models.User, error) {
	var users []models.User
	if err := uh.Base.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (uh *UserHandler) CreateUser(user *models.User) error {
	return uh.Base.DB.Create(user).Error
}

func (uh *UserHandler) GetUser(id string) (*models.User, error) {
	var user models.User
	if err := uh.Base.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (uh *UserHandler) UpdateUser(id string, updatedUser *models.User) error {
	var user models.User
	if err := uh.Base.DB.First(&user, id).Error; err != nil {
		return err
	}
	return uh.Base.DB.Model(&user).Updates(updatedUser).Error
}

func (uh *UserHandler) DeleteUser(id string) error {
	return uh.Base.DB.Delete(&models.User{}, id).Error
}
