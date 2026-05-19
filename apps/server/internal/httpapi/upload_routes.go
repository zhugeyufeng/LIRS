package httpapi

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func registerUploadRoutes(router *gin.Engine, api *gin.RouterGroup, repo repository) {
	uploadRoot := materialUploadRoot()
	router.StaticFS("/files", gin.Dir(uploadRoot, false))
	api.POST("/uploads/material-certificates", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles...); !ok {
			return
		}
		url, err := saveMaterialCertificateUpload(c, uploadRoot)
		respond(c, gin.H{"url": url}, err)
	})
}

func materialUploadRoot() string {
	uploadRoot := strings.TrimSpace(os.Getenv("UPLOAD_ROOT"))
	if uploadRoot == "" {
		return "uploads"
	}
	return uploadRoot
}

func saveMaterialCertificateUpload(c *gin.Context, uploadRoot string) (string, error) {
	const maxUploadBytes = 8 << 20

	if err := c.Request.ParseMultipartForm(maxUploadBytes); err != nil {
		return "", errors.New("标准品证书上传失败：文件超过 8MB 或表单无效")
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return "", errors.New("标准品证书上传失败：请选择证书文件")
	}
	defer file.Close()
	extension := strings.ToLower(filepath.Ext(header.Filename))
	if !validMaterialCertificateExtension(extension) {
		return "", errors.New("标准品证书上传失败：仅支持 PDF 文件")
	}
	if header.Size > maxUploadBytes {
		return "", errors.New("标准品证书上传失败：PDF 文件不能超过 8MB")
	}
	signature := make([]byte, 5)
	if _, err := io.ReadFull(file, signature); err != nil || string(signature) != "%PDF-" {
		return "", errors.New("标准品证书上传失败：文件内容不是有效 PDF")
	}
	dir := filepath.Join(uploadRoot, "material-certificates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("标准品证书上传失败：创建上传目录失败：%w", err)
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), extension)
	targetPath := filepath.Join(dir, filename)
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("标准品证书上传失败：保存证书失败：%w", err)
	}
	if _, err := target.Write(signature); err != nil {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("标准品证书上传失败：写入证书失败：%w", err)
	}
	written, err := io.Copy(target, io.LimitReader(file, maxUploadBytes-int64(len(signature))+1))
	if err != nil {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("标准品证书上传失败：写入证书失败：%w", err)
	}
	if written+int64(len(signature)) > maxUploadBytes {
		_ = target.Close()
		_ = os.Remove(targetPath)
		return "", errors.New("标准品证书上传失败：PDF 文件不能超过 8MB")
	}
	if err := target.Close(); err != nil {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("标准品证书上传失败：保存证书失败：%w", err)
	}
	return "/files/material-certificates/" + filename, nil
}

func validMaterialCertificateExtension(extension string) bool {
	switch extension {
	case ".pdf":
		return true
	default:
		return false
	}
}
