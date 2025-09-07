// Package main 启动应用程序
package main

import "github.com/yeisme/notevault/pkg/cmd"

//	@title			NoteVault API
//	@version		1.0
//	@description	NoteVault 是一个安全的笔记和文件存储服务，提供用户注册、登录、笔记管理和文件上传等功能。

//	@license.name	MIT
//	@license.url	https://opensource.org/license/mit/

//	@contact.name	yeisme
//	@contact.email	yefun2004@gmail.com.

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
