// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 images
package images

// import 多行导入块开始
import (
	// 语句或表达式
	"bytes"
	// 语句或表达式
	"image"
	// 语句或表达式
	"image/draw"
	// 语句或表达式
	"image/jpeg"
	// 语句或表达式
	_ "image/jpeg"
	// 语句或表达式
	_ "image/png"
	// 语句或表达式
	"io"
	// 闭合括号（import/var/参数列表等）
)

// 定义函数 CompressImage（签名开始）
func CompressImage(reader io.Reader) ([]byte, error) {
	// 短变量声明并赋值
	imageSrc, _, err := image.Decode(reader)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 短变量声明并赋值
	newImg := image.NewRGBA(imageSrc.Bounds())
	// 语句或表达式
	draw.Draw(newImg, newImg.Bounds(), imageSrc, imageSrc.Bounds().Min, draw.Src)
	// 语句或表达式
	draw.Draw(newImg, newImg.Bounds(), imageSrc, imageSrc.Bounds().Min, draw.Over)
	// 短变量声明并赋值
	buf := bytes.Buffer{}
	// 赋值语句
	err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 30})
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 返回语句
	return buf.Bytes(), err
	// 语句或表达式
}
