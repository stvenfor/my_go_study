package entity

// AuthorProfile 列表展示用作者信息（昵称 + 头像）。
//
// 原定义在 model/repository/community_repository.go 里，被社区与短视频两个域
// 的仓储接口同时消费（AuthorsByIDs / SearchUsers）。批 3 把两域切成垂直切片后，
// 仓储接口各自下沉到域包，该类型跨域不可见，故上提到跨域实体层。
type AuthorProfile struct {
	UserID   string
	Nickname string
	Avatar   string
}
