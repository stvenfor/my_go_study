// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 crypro
package crypro

// import 多行导入块开始
import (
	// 语句或表达式
	"crypto/aes"
	// 语句或表达式
	"crypto/cipher"
	// 语句或表达式
	"crypto/md5"
	// 语句或表达式
	"crypto/rand"
	// 语句或表达式
	"crypto/sha1"
	// 语句或表达式
	"encoding/base64"
	// 语句或表达式
	"encoding/hex"
	// 语句或表达式
	"errors"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"golang.org/x/crypto/bcrypt"
	// 语句或表达式
	"io"
	// 闭合括号（import/var/参数列表等）
)

// Md5WithSalt 使用 MD5 和 salt 进行哈希
// 定义函数 Md5WithSalt（签名开始）
func Md5WithSalt(data, salt string) string {
	// 将 salt 和 data 组合
	// 短变量声明并赋值
	combined := data + salt
	// 生成 MD5 哈希
	// 短变量声明并赋值
	hash := md5.Sum([]byte(combined))
	// 转为十六进制字符串
	// 返回语句
	return hex.EncodeToString(hash[:])
	// 语句或表达式
}

// 定义函数 Md5（签名开始）
func Md5(data []byte) string {
	// 生成 MD5 哈希
	// 短变量声明并赋值
	hash := md5.Sum(data)
	// 转为十六进制字符串
	// 返回语句
	return hex.EncodeToString(hash[:])
	// 语句或表达式
}

// HashPassword 使用 bcrypt 生成哈希密码
// 定义函数 HashPassword（签名开始）
func HashPassword(password string) (string, error) {
	// 使用默认成本值生成哈希
	// 短变量声明并赋值
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 返回语句
	return string(hashedBytes), nil
	// 语句或表达式
}

// CheckPasswordHash 验证密码是否匹配
// 定义函数 CheckPasswordHash（签名开始）
func CheckPasswordHash(password, hash string) bool {
	// 短变量声明并赋值
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	// 返回语句
	return err == nil
	// 语句或表达式
}

// 定义函数 Sha1（签名开始）
func Sha1(input string) string {
	// 短变量声明并赋值
	hash := sha1.New()
	// 写入数据到哈希器
	// 语句或表达式
	hash.Write([]byte(input))
	// 计算哈希值
	// 短变量声明并赋值
	hashBytes := hash.Sum(nil)
	// 将哈希值转换为十六进制字符串
	// 短变量声明并赋值
	hashString := hex.EncodeToString(hashBytes)
	// 返回语句
	return hashString
	// 语句或表达式
}

// 定义函数 encodeBase64（签名开始）
func encodeBase64(input []byte) string {
	// 返回语句
	return base64.URLEncoding.EncodeToString(input)
	// 语句或表达式
}

// 使用 URL Safe Base64 解码
// 定义函数 decodeBase64（签名开始）
func decodeBase64(input string) ([]byte, error) {
	// 返回语句
	return base64.URLEncoding.DecodeString(input)
	// 语句或表达式
}

// 常量声明
const Slat = "mscommunity!@#$%"

// EncryptString 加密字符串
// 定义函数 EncryptString（签名开始）
func EncryptString(key, plaintext string) (string, error) {
	// 将 key 转换为字节数组
	// 短变量声明并赋值
	keyBytes := []byte(key)
	// 条件判断 if
	if len(keyBytes) != 16 {
		// 返回语句
		return "", errors.New("key 长度必须为 16 字节")
		// 语句或表达式
	}

	// 创建 AES 加密块
	// 短变量声明并赋值
	block, err := aes.NewCipher(keyBytes)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("创建 AES 块失败: %w", err)
		// 语句或表达式
	}

	// 初始化随机化向量 (IV)
	// 短变量声明并赋值
	iv := make([]byte, aes.BlockSize)
	// 条件判断 if
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		// 返回语句
		return "", fmt.Errorf("生成 IV 失败: %w", err)
		// 语句或表达式
	}

	// 创建加密器
	// 短变量声明并赋值
	stream := cipher.NewCFBEncrypter(block, iv)
	// 短变量声明并赋值
	ciphertext := make([]byte, len(plaintext))
	// 语句或表达式
	stream.XORKeyStream(ciphertext, []byte(plaintext))

	// 将 IV 和密文组合后返回 Base64 编码
	// 短变量声明并赋值
	final := append(iv, ciphertext...)
	// 返回语句
	return encodeBase64(final), nil
	// 语句或表达式
}

// 解密字符串
// 定义函数 DecryptString（签名开始）
func DecryptString(key, ciphertext string) (string, error) {
	// 将 key 转换为字节数组
	// 短变量声明并赋值
	keyBytes := []byte(key)
	// 条件判断 if
	if len(keyBytes) != 16 {
		// 返回语句
		return "", errors.New("key 长度必须为 16 字节")
		// 语句或表达式
	}

	// Base64 解码
	// 短变量声明并赋值
	data, err := decodeBase64(ciphertext)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("Base64 解码失败: %w", err)
		// 语句或表达式
	}

	// 提取 IV 和密文
	// 条件判断 if
	if len(data) < aes.BlockSize {
		// 返回语句
		return "", errors.New("密文长度不足")
		// 语句或表达式
	}
	// 短变量声明并赋值
	iv := data[:aes.BlockSize]
	// 短变量声明并赋值
	ciphertextBytes := data[aes.BlockSize:]

	// 创建 AES 解密块
	// 短变量声明并赋值
	block, err := aes.NewCipher(keyBytes)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", fmt.Errorf("创建 AES 块失败: %w", err)
		// 语句或表达式
	}

	// 创建解密器
	// 短变量声明并赋值
	stream := cipher.NewCFBDecrypter(block, iv)
	// 短变量声明并赋值
	plaintext := make([]byte, len(ciphertextBytes))
	// 语句或表达式
	stream.XORKeyStream(plaintext, ciphertextBytes)

	// 返回语句
	return string(plaintext), nil
	// 语句或表达式
}
