package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateVerificationCode() string {
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000)
}

// 可以添加更多通用工具函数
