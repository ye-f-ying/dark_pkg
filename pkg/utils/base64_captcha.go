package utils

import (
	"image/color"

	"github.com/mojocn/base64Captcha"
)

// MathCaptchaResult 生成数学公式验证码返回结果
type MathCaptcha struct {
	ImageBase64 string // 图片base64（data:image/png;base64,xxx）
	Answer      string // 答案
}

func GenerateMathCaptcha() (*MathCaptcha, error) {
	// 配置：数学公式型
	driver := &base64Captcha.DriverMath{
		Height:          50,
		Width:           130,
		NoiseCount:      2,
		ShowLineOptions: base64Captcha.OptionShowHollowLine,
		BgColor: &color.RGBA{
			R: 248, G: 248, B: 248, A: 255,
		},
	}

	c := base64Captcha.NewCaptcha(driver, base64Captcha.NewMemoryStore(0, 0))

	// 生成
	_, b64, answer, err := c.Generate()
	if err != nil {
		return nil, err
	}

	return &MathCaptcha{
		ImageBase64: b64,
		Answer:      answer,
	}, nil
}
