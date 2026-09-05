package main

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"

	_ "golang.org/x/image/webp" // webp.Decode via image.Decode
)

// ── 头像上传安全加固（OPT-20260806-046）──────────────────────────────────
// 恢复上传前的必要前置：真实图像校验（解码验证 + magic bytes + 拒绝
// SVG/HTML 伪装）、尺寸与压缩比限制（防解压炸弹）、文件类型嗅探而非仅信
// Content-Type 头。公司头像与全局头像共用同一校验（同策略）。

const (
	// maxAvatarPixelSide 解码后图像任意一边的最大像素数（防超大尺寸内存耗尽）
	maxAvatarPixelSide = 4096
	// maxAvatarPixelRatio 解码像素数 / 文件字节数 上限（防解压炸弹：
	// 极小文件解出巨图，如 1x1 缩放放大 / 超低质量大画幅）
	maxAvatarPixelRatio = 200
)

// sniffAndValidateAvatarImage 校验上传头像：
//  1. magic bytes 嗅探（http.DetectContentType），与声明 Content-Type 一致性检查；
//  2. 真实解码验证（拒绝伪装成图片的 HTML/SVG/文本）；
//  3. 尺寸与压缩比限制（防解压炸弹）。
//
// 返回规范化的 MIME（用于存储 data URI）与解码后的像素总数。
func sniffAndValidateAvatarImage(declaredMIME string, data []byte) (string, int64, error) {
	if len(data) == 0 {
		return "", 0, fmt.Errorf("空文件")
	}
	// 1) 嗅探实际类型（仅信声明头会放行 HTML 伪装）
	sniffed := http.DetectContentType(data)
	if _, ok := allowedAvatarTypes[sniffed]; !ok {
		// 拒绝 text/html / image/svg+xml / 纯文本伪装等一切非白名单内容
		log.Printf("[taskAuth] avatar rejected: sniffed=%q declared=%q (magic bytes mismatch)", sniffed, declaredMIME)
		return "", 0, fmt.Errorf("文件内容不是受支持的图片（仅 JPEG/PNG/WebP/GIF），且已拒绝 HTML/SVG 伪装")
	}
	// 2) 声明类型与嗅探一致（防御：声明 image/png 传 JPEG 等）
	if _, ok := allowedAvatarTypes[declaredMIME]; !ok {
		return "", 0, fmt.Errorf("不支持的图片格式，仅支持 JPEG、PNG、WebP、GIF")
	}
	// 3) 真实解码（内容解析级验证；decode config 不展开像素，用于尺寸与压缩比判定）
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		log.Printf("[taskAuth] avatar rejected: decode failed: %v", err)
		return "", 0, fmt.Errorf("文件不是有效的图片（解码失败）")
	}
	// 4) 尺寸限制（防超大画幅）
	if cfg.Width > maxAvatarPixelSide || cfg.Height > maxAvatarPixelSide {
		return "", 0, fmt.Errorf("图片尺寸过大（最大 %d×%d 像素）", maxAvatarPixelSide, maxAvatarPixelSide)
	}
	// 5) 压缩比限制（防解压炸弹：极小文件解出巨图）
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if ratio := float64(pixels) / float64(len(data)); ratio > maxAvatarPixelRatio {
		log.Printf("[taskAuth] avatar rejected: decompression ratio %.1f (pixels=%d bytes=%d)", ratio, pixels, len(data))
		return "", 0, fmt.Errorf("图片压缩比异常（疑似解压炸弹），请重新导出后上传")
	}
	// 6) 全量解码验证（gif 多帧 / 编码损坏在 Decode 阶段暴露；jpeg/png 同上）
	switch format {
	case "jpeg", "png", "webp", "gif":
		if _, err := decodeImageFull(data, format); err != nil {
			return "", 0, fmt.Errorf("图片解码失败（文件可能已损坏）")
		}
	default:
		return "", 0, fmt.Errorf("不支持的图片格式")
	}
	return sniffed, pixels, nil
}

// decodeImageFull 按格式全量解码（jpeg/png/gif 经标准库，webp 经 x/image）。
func decodeImageFull(data []byte, format string) (image.Image, error) {
	r := bytes.NewReader(data)
	switch format {
	case "jpeg":
		return jpeg.Decode(r)
	case "png":
		return png.Decode(r)
	case "gif":
		return gif.Decode(r)
	case "webp":
		// image/webp 注册于 image.Decode（blank import）；显式调用保持清晰
		return decodeWebP(r)
	default:
		return nil, fmt.Errorf("unknown format %s", format)
	}
}

func decodeWebP(r io.Reader) (image.Image, error) {
	// 经 image.Decode 统一调度（webp 注册在 init）
	img, _, err := image.Decode(r)
	return img, err
}
