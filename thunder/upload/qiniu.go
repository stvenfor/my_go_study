// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 upload
package upload

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"io"
	// 语句或表达式
	"net/url"
	// 语句或表达式
	"strings"

	// 语句或表达式
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	// 语句或表达式
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	// 语句或表达式
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	// 语句或表达式
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	// 语句或表达式
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 QiniuUpload
type QiniuUpload struct {
	// 语句或表达式
	uploadManager *uploader.UploadManager
	// 语句或表达式
	bucket *objects.Bucket
	// 语句或表达式
	credentials *credentials.Credentials
	// 语句或表达式
}

// 定义函数 InitQiniuUpload（签名开始）
func InitQiniuUpload(regionId, bucket, accessKey, secretKey string) (*QiniuUpload, error) {
	// 短变量声明并赋值
	mac := credentials.NewCredentials(accessKey, secretKey)
	// 短变量声明并赋值
	clientOptions := http_client.Options{
		// 列表项（字段/参数/元素），行末逗号
		Credentials: mac,
		// 列表项（字段/参数/元素），行末逗号
		Regions: region.GetRegionByID(regionId, true),
		// 语句或表达式
	}

	// 短变量声明并赋值
	uploadManager := uploader.NewUploadManager(&uploader.UploadManagerOptions{
		// 列表项（字段/参数/元素），行末逗号
		Options: clientOptions,
		// 语句或表达式
	})
	// 短变量声明并赋值
	objectsManager := objects.NewObjectsManager(&objects.ObjectsManagerOptions{
		// 列表项（字段/参数/元素），行末逗号
		Options: clientOptions,
		// 语句或表达式
	})
	// 短变量声明并赋值
	b := objectsManager.Bucket(bucket)
	// 短变量声明并赋值
	qiniuUploadManager := &QiniuUpload{
		// 列表项（字段/参数/元素），行末逗号
		uploadManager: uploadManager,
		// 列表项（字段/参数/元素），行末逗号
		bucket: b,
		// 列表项（字段/参数/元素），行末逗号
		credentials: mac,
		// 语句或表达式
	}
	// 返回语句
	return qiniuUploadManager, nil
	// 语句或表达式
}

// 定义函数 Upload（签名开始）
func (q *QiniuUpload) Upload(ctx context.Context, bucket string, reader io.Reader, name string) error {
	// 短变量声明并赋值
	err := q.uploadManager.UploadReader(ctx, reader, &uploader.ObjectOptions{
		// 列表项（字段/参数/元素），行末逗号
		BucketName: bucket,
		// 列表项（字段/参数/元素），行末逗号
		ObjectName: &name,
		// 列表项（字段/参数/元素），行末逗号
		FileName: name,
		// 语句或表达式
	}, nil)
	// 返回语句
	return err
	// 语句或表达式
}

// Delete 从七牛云存储中删除指定的文件
// 定义函数 Delete（签名开始）
func (q *QiniuUpload) Delete(ctx context.Context, key string) error {
	// 返回语句
	return q.bucket.Object(key).Delete().Call(ctx)
	// 语句或表达式
}

// GetPublicURL 获取文件的公开访问URL
// 定义函数 GetPublicURL（签名开始）
func (q *QiniuUpload) GetPublicURL(domain string, key string) string {
	// 构建公开访问的URL
	// 条件判断 if
	if strings.HasPrefix(domain, "http") {
		// 返回语句
		return fmt.Sprintf("%s/%s", domain, key)
		// 语句或表达式
	}
	// 短变量声明并赋值
	u := url.URL{
		// 列表项（字段/参数/元素），行末逗号
		Scheme: "https",
		// 列表项（字段/参数/元素），行末逗号
		Host: domain,
		// 列表项（字段/参数/元素），行末逗号
		Path: key,
		// 语句或表达式
	}
	// 返回语句
	return u.String()
	// 语句或表达式
}
