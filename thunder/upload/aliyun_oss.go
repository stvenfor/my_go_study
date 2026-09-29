// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 upload
package upload

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"errors"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"io"
	// 语句或表达式
	"strings"

	// 语句或表达式
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	// 语句或表达式
	"github.com/mszlu521/thunder/logs"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 AliyunOSSUpload
type AliyunOSSUpload struct {
	// 语句或表达式
	client *oss.Client
	// 语句或表达式
	bucket *oss.Bucket
	// 语句或表达式
	accessKeyID string
	// 语句或表达式
	accessKeySecret string
	// 语句或表达式
	endpoint string
	// 语句或表达式
	bucketName string
	// 语句或表达式
}

// InitAliyunOSSUpload 初始化阿里云OSS上传管理器
// 定义函数 InitAliyunOSSUpload（签名开始）
func InitAliyunOSSUpload(accessKeyID, accessKeySecret, endpoint, bucketName string) (*AliyunOSSUpload, error) {
	// 检查必要配置是否存在
	// 条件判断 if
	if accessKeyID == "" || accessKeySecret == "" || endpoint == "" || bucketName == "" {
		// 语句或表达式
		logs.Info("阿里云OSS配置不完整，跳过初始化")
		// 返回语句
		return nil, errors.New("阿里云OSS配置不完整")
		// 语句或表达式
	}

	// 创建OSSClient实例
	// 短变量声明并赋值
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("创建阿里云OSS客户端失败: %v", err)
		// 返回语句
		return nil, err
		// 语句或表达式
	}

	// 获取存储空间
	// 短变量声明并赋值
	bucket, err := client.Bucket(bucketName)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("获取阿里云OSS存储空间失败: %v", err)
		// 返回语句
		return nil, err
		// 语句或表达式
	}

	// 短变量声明并赋值
	aliyunOSSUploadManager := &AliyunOSSUpload{
		// 列表项（字段/参数/元素），行末逗号
		client: client,
		// 列表项（字段/参数/元素），行末逗号
		bucket: bucket,
		// 列表项（字段/参数/元素），行末逗号
		accessKeyID: accessKeyID,
		// 列表项（字段/参数/元素），行末逗号
		accessKeySecret: accessKeySecret,
		// 列表项（字段/参数/元素），行末逗号
		endpoint: endpoint,
		// 列表项（字段/参数/元素），行末逗号
		bucketName: bucketName,
		// 语句或表达式
	}

	// 语句或表达式
	logs.Info("阿里云OSS上传服务初始化成功")
	// 返回语句
	return aliyunOSSUploadManager, nil
	// 语句或表达式
}

// Upload 上传文件到阿里云OSS
// 定义函数 Upload（签名开始）
func (a *AliyunOSSUpload) Upload(ctx context.Context, reader io.Reader, name string) error {
	// 条件判断 if
	if a == nil || a.bucket == nil {
		// 返回语句
		return errors.New("no such bucket")
		// 语句或表达式
	}

	// 直接使用PutObject上传
	// 短变量声明并赋值
	err := a.bucket.PutObject(name, reader)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("上传文件到阿里云OSS失败: %v", err)
		// 返回语句
		return err
		// 语句或表达式
	}

	// 返回语句
	return nil
	// 语句或表达式
}

// UploadWithMetadata 上传文件到阿里云OSS并附带元数据
// 定义函数 UploadWithMetadata（签名开始）
func (a *AliyunOSSUpload) UploadWithMetadata(ctx context.Context, reader io.Reader, name string, metadata map[string]string) error {
	// 条件判断 if
	if a == nil || a.bucket == nil {
		// 返回语句
		return errors.New("no such bucket")
		// 语句或表达式
	}

	// 设置对象元数据
	// 变量声明
	var options []oss.Option
	// for 循环
	for key, value := range metadata {
		// 赋值语句
		options = append(options, oss.Meta(key, value))
		// 语句或表达式
	}

	// 上传对象
	// 短变量声明并赋值
	err := a.bucket.PutObject(name, reader, options...)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("上传文件到阿里云OSS失败: %v", err)
		// 返回语句
		return err
		// 语句或表达式
	}

	// 返回语句
	return nil
	// 语句或表达式
}

// GetSignedURL 获取对象的签名URL，用于临时访问
// 定义函数 GetSignedURL（签名开始）
func (a *AliyunOSSUpload) GetSignedURL(objectKey string, expiredInSec int64) (string, error) {
	// 条件判断 if
	if a == nil || a.bucket == nil {
		// 返回语句
		return "", errors.New("no such bucket")
		// 语句或表达式
	}

	// 生成签名URL
	// 短变量声明并赋值
	signedURL, err := a.bucket.SignURL(objectKey, oss.HTTPGet, expiredInSec)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("生成阿里云OSS签名URL失败: %v", err)
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 返回语句
	return signedURL, nil
	// 语句或表达式
}

// DeleteObject 删除OSS中的对象
// 定义函数 DeleteObject（签名开始）
func (a *AliyunOSSUpload) DeleteObject(objectKey string) error {
	// 条件判断 if
	if a == nil || a.bucket == nil {
		// 返回语句
		return errors.New("no such bucket")
		// 语句或表达式
	}

	// 短变量声明并赋值
	err := a.bucket.DeleteObject(objectKey)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		logs.Errorf("删除阿里云OSS对象失败: %v", err)
		// 返回语句
		return err
		// 语句或表达式
	}

	// 返回语句
	return nil
	// 语句或表达式
}

// IsAvailable 检查阿里云OSS服务是否可用
// 定义函数 IsAvailable（签名开始）
func (a *AliyunOSSUpload) IsAvailable() bool {
	// 返回语句
	return a != nil && a.bucket != nil
	// 语句或表达式
}

// GetObjectURL 获取对象的访问URL
// 定义函数 GetObjectURL（签名开始）
func (a *AliyunOSSUpload) GetObjectURL(objectKey string) string {
	// 条件判断 if
	if a == nil || a.bucket == nil {
		// 返回语句
		return ""
		// 语句或表达式
	}

	// 处理endpoint，确保格式正确
	// 条件判断 if
	if !strings.HasPrefix(a.endpoint, "http") {
		// 语句或表达式
		a.endpoint = "https://" + a.endpoint
		// 语句或表达式
	}

	// 构造对象URL
	// 短变量声明并赋值
	objectURL := a.endpoint + "/" + a.bucketName + "/" + objectKey
	// 返回语句
	return objectURL
	// 语句或表达式
}

// 定义函数 GetPublicUrl（签名开始）
func (a *AliyunOSSUpload) GetPublicUrl(filename string) string {
	// 返回语句
	return fmt.Sprintf("https://%s.%s/%s", a.bucketName, a.endpoint, filename)
	// 语句或表达式
}
